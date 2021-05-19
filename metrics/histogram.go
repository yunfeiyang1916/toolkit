package metrics

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	metrics "github.com/rcrowley/go-metrics"
)

var (
	DefBuckets []int
	// nolint:lll
	defBuckets = []float64{0.0005, 0.001, 0.002, 0.003, 0.004, 0.005, 0.006, 0.007, 0.008, 0.009, 0.01, 0.02, 0.03, 0.04, 0.05, 0.06, 0.07, 0.08, 0.09, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 25, 30, 35, 40, 45, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 175, 200, 225, 250, 300, 350, 400, 450, 500, 600, 700, 800, 900, 1000, 1500, 2000, 2500, 3000, 4000, 5000, 7500, 10000, 15000, 20000, 25000, 30000, 40000, 50000, 60000}
)

func init() {
	DefBuckets = make([]int, len(defBuckets))
	for i, b := range defBuckets {
		DefBuckets[i] = int(float64(time.Millisecond) * b)
	}
}

func newHistogram(upperBounds []int) metrics.Histogram {
	h := &histogram{
		upperBounds: upperBounds,
		counts:      [2]*histogramCounts{&histogramCounts{}, &histogramCounts{}},
	}
	if h.upperBounds == nil {
		h.upperBounds = DefBuckets
	}
	for i, upperBound := range h.upperBounds {
		if i < len(h.upperBounds)-1 {
			if upperBound >= h.upperBounds[i+1] {
				// nolint:iklint
				panic(fmt.Errorf(
					"histogram buckets must be in increasing order: %d >= %d",
					upperBound, h.upperBounds[i+1],
				))
			}
		}
	}
	// Finally we know the final length of h.upperBounds and can make buckets
	// for both counts:
	h.counts[0].buckets = make([]uint64, len(h.upperBounds))
	h.counts[1].buckets = make([]uint64, len(h.upperBounds))
	return h
}

type histogramCounts struct {
	// sumBits contains the bits of the float64 representing the sum of all
	// observations. sumBits and count have to go first in the struct to
	// guarantee alignment for atomic operations.
	// http://golang.org/pkg/sync/atomic/#pkg-note-BUG
	sum     int64
	count   uint64
	buckets []uint64
}

type histogram struct {
	// countAndHotIdx is a complicated one. For lock-free yet atomic
	// observations, we need to save the total count of observations again,
	// combined with the index of the currently-hot counts struct, so that
	// we can perform the operation on both values atomically. The least
	// significant bit defines the hot counts struct. The remaining 63 bits
	// represent the total count of observations. This happens under the
	// assumption that the 63bit count will never overflow. Rationale: An
	// observations takes about 30ns. Let's assume it could happen in
	// 10ns. Overflowing the counter will then take at least (2^63)*10ns,
	// which is about 3000 years.
	//
	// This has to be first in the struct for 64bit alignment. See
	// http://golang.org/pkg/sync/atomic/#pkg-note-BUG
	countAndHotIdx uint64

	writeMtx sync.Mutex // Only used in the Snapshot method.

	upperBounds []int

	// Two counts, one is "hot" for lock-free observations, the other is
	// "cold" for writing out a dto.Metric. It has to be an array of
	// pointers to guarantee 64bit alignment of the histogramCounts, see
	// http://golang.org/pkg/sync/atomic/#pkg-note-BUG.
	counts [2]*histogramCounts
	hotIdx int // Index of currently-hot counts. Only used within Write.

	lastSummary *histogramSummary
}

// Clear clears the histogram and its sample.
func (h *histogram) Clear() {}

// Count returns the number of samples recorded since the histogram was last
// cleared.
func (h *histogram) Count() int64 { return 0 }

// Max returns the maximum value in the sample.
func (h *histogram) Max() int64 { return 0 }

// Mean returns the mean of the values in the sample.
func (h *histogram) Mean() float64 { return 0 }

// Min returns the minimum value in the sample.
func (h *histogram) Min() int64 { return 0 }

// Percentile returns an arbitrary percentile of the values in the sample.
func (h *histogram) Percentile(p float64) float64 { return 0 }

// Percentiles returns a slice of arbitrary percentiles of the values in the
// sample.
func (h *histogram) Percentiles(ps []float64) []float64 { return nil }

// Sample returns the Sample underlying the histogram.
func (h *histogram) Sample() metrics.Sample { return nil }

// StdDev returns the standard deviation of the values in the sample.
func (h *histogram) StdDev() float64 { return 0 }

// Sum returns the sum in the sample.
func (h *histogram) Sum() int64 { return 0 }

// Variance returns the variance of the values in the sample.
func (h *histogram) Variance() float64 { return 0 }

func (h *histogram) Update(v int64) {
	// TODO(beorn7): For small numbers of buckets (<30), a linear search is
	// slightly faster than the binary search. If we really care, we could
	// switch from one search strategy to the other depending on the number
	// of buckets.
	//
	// Microbenchmarks (BenchmarkHistogramNoLabels):
	// 11 buckets: 38.3 ns/op linear - binary 48.7 ns/op
	// 100 buckets: 78.1 ns/op linear - binary 54.9 ns/op
	// 300 buckets: 154 ns/op linear - binary 61.6 ns/op
	i := sort.SearchInts(h.upperBounds, int(v))

	// We increment h.countAndHotIdx by 2 so that the counter in the upper
	// 63 bits gets incremented by 1. At the same time, we get the new value
	// back, which we can use to find the currently-hot counts.
	n := atomic.AddUint64(&h.countAndHotIdx, 2)
	hotCounts := h.counts[n%2]

	if i < len(h.upperBounds) {
		atomic.AddUint64(&hotCounts.buckets[i], 1)
	}
	atomic.AddInt64(&hotCounts.sum, v)
	atomic.AddUint64(&hotCounts.count, 1)
}

type histogramBucket struct {
	Count      uint64
	UpperBound int
}

type histogramSummary struct {
	SampleCount     uint64
	SampleSum       int64
	Bucket          []*histogramBucket
	DiffBucket      []*histogramBucket
	LastSampleCount uint64
	LastSampleSum   int64
}

func (h *histogram) Snapshot() metrics.Histogram {
	var (
		his                   = &histogramSummary{}
		buckets               = make([]*histogramBucket, len(h.upperBounds))
		hotCounts, coldCounts *histogramCounts
		count                 uint64
	)

	// For simplicity, we mutex the rest of this method. It is not in the
	// hot path, i.e.  Observe is called much more often than Write. The
	// complication of making Write lock-free isn't worth it.
	h.writeMtx.Lock()
	defer h.writeMtx.Unlock()

	// This is a bit arcane, which is why the following spells out this if
	// clause in English:
	//
	// If the currently-hot counts struct is #0, we atomically increment
	// h.countAndHotIdx by 1 so that from now on Observe will use the counts
	// struct #1. Furthermore, the atomic increment gives us the new value,
	// which, in its most significant 63 bits, tells us the count of
	// observations done so far up to and including currently ongoing
	// observations still using the counts struct just changed from hot to
	// cold. To have a normal uint64 for the count, we bitshift by 1 and
	// save the result in count. We also set h.hotIdx to 1 for the next
	// Write call, and we will refer to counts #1 as hotCounts and to counts
	// #0 as coldCounts.
	//
	// If the currently-hot counts struct is #1, we do the corresponding
	// things the other way round. We have to _decrement_ h.countAndHotIdx
	// (which is a bit arcane in itself, as we have to express -1 with an
	// unsigned int...).
	if h.hotIdx == 0 {
		count = atomic.AddUint64(&h.countAndHotIdx, 1) >> 1
		h.hotIdx = 1
		hotCounts = h.counts[1]
		coldCounts = h.counts[0]
	} else {
		count = atomic.AddUint64(&h.countAndHotIdx, ^uint64(0)) >> 1 // Decrement.
		h.hotIdx = 0
		hotCounts = h.counts[0]
		coldCounts = h.counts[1]
	}

	// Now we have to wait for the now-declared-cold counts to actually cool
	// down, i.e. wait for all observations still using it to finish. That's
	// the case once the count in the cold counts struct is the same as the
	// one atomically retrieved from the upper 63bits of h.countAndHotIdx.
	for {
		if count == atomic.LoadUint64(&coldCounts.count) {
			break
		}
		runtime.Gosched() // Let observations get work done.
	}

	his.SampleCount = count
	his.SampleSum = atomic.LoadInt64(&coldCounts.sum)
	for i, upperBound := range h.upperBounds {
		buckets[i] = &histogramBucket{
			Count:      atomic.LoadUint64(&coldCounts.buckets[i]),
			UpperBound: upperBound,
		}
	}

	his.Bucket = buckets
	if h.lastSummary == nil {
		his.DiffBucket = buckets
	} else {
		his.LastSampleSum = h.lastSummary.SampleSum
		his.LastSampleCount = h.lastSummary.SampleCount
		his.DiffBucket = make([]*histogramBucket, len(h.upperBounds))
		for i := range h.upperBounds {
			his.DiffBucket[i] = &histogramBucket{
				Count:      buckets[i].Count - h.lastSummary.Bucket[i].Count,
				UpperBound: h.upperBounds[i],
			}
		}
	}

	// Finally add all the cold counts to the new hot counts and reset the cold counts.
	atomic.AddUint64(&hotCounts.count, count)
	atomic.StoreUint64(&coldCounts.count, 0)
	atomic.AddInt64(&hotCounts.sum, his.SampleSum)
	atomic.StoreInt64(&coldCounts.sum, 0)
	for i := range h.upperBounds {
		atomic.AddUint64(&hotCounts.buckets[i], atomic.LoadUint64(&coldCounts.buckets[i]))
		atomic.StoreUint64(&coldCounts.buckets[i], 0)
	}
	h.lastSummary = his
	return his
}

// nolint:iklint
func (h *histogramSummary) Clear() { panic("Clear called on a Snapshot") }

// nolint:iklint
func (h *histogramSummary) Update(int64) { panic("Update called on a Snapshot") }

func (h *histogramSummary) Count() int64 { return int64(h.SampleCount) }

func (h *histogramSummary) Mean() float64 {
	total := h.SampleSum - h.LastSampleSum
	count := h.SampleCount - h.LastSampleCount
	if count == 0 {
		return 0
	}
	return float64(total) / float64(count)
}

func (h *histogramSummary) Max() int64 {
	l := len(h.DiffBucket)
	for i := l - 1; i >= 0; i-- {
		if h.DiffBucket[i].Count > 0 {
			return int64(h.DiffBucket[i].UpperBound)
		}
	}
	return 0
}

func (h *histogramSummary) Min() int64 {
	for _, b := range h.DiffBucket {
		if b.Count > 0 {
			return int64(b.UpperBound)
		}
	}
	return 0
}

func (h *histogramSummary) Percentile(p float64) float64 {
	return h.Percentiles([]float64{p})[0]
}

func (h *histogramSummary) Percentiles(ps []float64) []float64 {
	bucketLen := len(h.DiffBucket)
	count := make([]int, bucketLen)
	for i, b := range h.DiffBucket {
		if i == 0 {
			count[i] = int(b.Count)
		} else {
			count[i] = int(b.Count) + count[i-1]
		}
	}
	scores := make([]float64, len(ps))
	for i, p := range ps {
		pCount := int(float64(count[bucketLen-1]+1) * p)
		index := sort.SearchInts(count, pCount)
		if index == 0 {
			scores[i] = float64(h.DiffBucket[0].UpperBound)
		} else if index >= bucketLen {
			scores[i] = float64(h.DiffBucket[bucketLen-1].UpperBound)
		} else {
			scores[i] = float64(h.DiffBucket[index].UpperBound)
		}
	}
	return scores
}

func (h *histogramSummary) Sample() metrics.Sample { return nil }

func (h *histogramSummary) Snapshot() metrics.Histogram { return h }

func (h *histogramSummary) StdDev() float64 { return math.Sqrt(h.Variance()) }

func (h *histogramSummary) Sum() int64 { return h.SampleSum }

func (h *histogramSummary) Variance() float64 {
	m := h.Mean()
	var sum float64
	var total uint64
	for _, b := range h.DiffBucket {
		d := (float64(b.UpperBound) - m)
		sum = (d * d) * float64(b.Count)
		total += b.Count
	}
	return sum / float64(total)
}

// type buckSort []*histogramBucket
//
// func (s buckSort) Len() int {
// 	return len(s)
// }
//
// func (s buckSort) Swap(i, j int) {
// 	s[i], s[j] = s[j], s[i]
// }
//
// func (s buckSort) Less(i, j int) bool {
// 	return s[i].UpperBound < s[j].UpperBound
// }
