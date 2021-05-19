package metrics

import (
	"sync"
	"time"

	metrics "github.com/rcrowley/go-metrics"
)

func NewTimer() metrics.Timer {
	return &StandardTimer{
		histogram: newHistogram(nil),
		meter:     metrics.NewMeter(),
	}
}

// StandardTimer is the standard implementation of a Timer and uses a Histogram
// and Meter.
type StandardTimer struct {
	histogram metrics.Histogram
	meter     metrics.Meter
	mutex     sync.Mutex
}

// Count returns the number of events recorded.
func (t *StandardTimer) Count() int64 {
	return t.histogram.Count()
}

// Max returns the maximum value in the sample.
func (t *StandardTimer) Max() int64 {
	return t.histogram.Max()
}

// Mean returns the mean of the values in the sample.
func (t *StandardTimer) Mean() float64 {
	return t.histogram.Mean()
}

// Min returns the minimum value in the sample.
func (t *StandardTimer) Min() int64 {
	return t.histogram.Min()
}

// Percentile returns an arbitrary percentile of the values in the sample.
func (t *StandardTimer) Percentile(p float64) float64 {
	return t.histogram.Percentile(p)
}

// Percentiles returns a slice of arbitrary percentiles of the values in the
// sample.
func (t *StandardTimer) Percentiles(ps []float64) []float64 {
	return t.histogram.Percentiles(ps)
}

// Rate1 returns the one-minute moving average rate of events per second.
func (t *StandardTimer) Rate1() float64 {
	return t.meter.Rate1()
}

// Rate5 returns the five-minute moving average rate of events per second.
func (t *StandardTimer) Rate5() float64 {
	return t.meter.Rate5()
}

// Rate15 returns the fifteen-minute moving average rate of events per second.
func (t *StandardTimer) Rate15() float64 {
	return t.meter.Rate15()
}

// RateMean returns the meter's mean rate of events per second.
func (t *StandardTimer) RateMean() float64 {
	return t.meter.RateMean()
}

// Snapshot returns a read-only copy of the timer.
func (t *StandardTimer) Snapshot() metrics.Timer {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return &TimerSnapshot{
		histogram: t.histogram.Snapshot(),
		meter:     t.meter.Snapshot(),
	}
}

// StdDev returns the standard deviation of the values in the sample.
func (t *StandardTimer) StdDev() float64 {
	return t.histogram.StdDev()
}

// Stop stops the meter.
func (t *StandardTimer) Stop() {
	t.meter.Stop()
}

// Sum returns the sum in the sample.
func (t *StandardTimer) Sum() int64 {
	return t.histogram.Sum()
}

// Record the duration of the execution of the given function.
func (t *StandardTimer) Time(f func()) {
	ts := time.Now()
	f()
	t.Update(time.Since(ts))
}

// Record the duration of an event.
func (t *StandardTimer) Update(d time.Duration) {
	t.histogram.Update(int64(d))
	t.meter.Mark(1)
}

// Record the duration of an event that started at a time and ends now.
func (t *StandardTimer) UpdateSince(ts time.Time) {
	t.histogram.Update(int64(time.Since(ts)))
	t.meter.Mark(1)
}

// Variance returns the variance of the values in the sample.
func (t *StandardTimer) Variance() float64 {
	return t.histogram.Variance()
}

// TimerSnapshot is a read-only copy of another Timer.
type TimerSnapshot struct {
	histogram metrics.Histogram
	meter     metrics.Meter
}

// Count returns the number of events recorded at the time the snapshot was
// taken.
func (t *TimerSnapshot) Count() int64 { return t.histogram.Count() }

// Max returns the maximum value at the time the snapshot was taken.
func (t *TimerSnapshot) Max() int64 { return t.histogram.Max() }

// Mean returns the mean value at the time the snapshot was taken.
func (t *TimerSnapshot) Mean() float64 { return t.histogram.Mean() }

// Min returns the minimum value at the time the snapshot was taken.
func (t *TimerSnapshot) Min() int64 { return t.histogram.Min() }

// Percentile returns an arbitrary percentile of sampled values at the time the
// snapshot was taken.
func (t *TimerSnapshot) Percentile(p float64) float64 {
	return t.histogram.Percentile(p)
}

// Percentiles returns a slice of arbitrary percentiles of sampled values at
// the time the snapshot was taken.
func (t *TimerSnapshot) Percentiles(ps []float64) []float64 {
	return t.histogram.Percentiles(ps)
}

// Rate1 returns the one-minute moving average rate of events per second at the
// time the snapshot was taken.
func (t *TimerSnapshot) Rate1() float64 { return t.meter.Rate1() }

// Rate5 returns the five-minute moving average rate of events per second at
// the time the snapshot was taken.
func (t *TimerSnapshot) Rate5() float64 { return t.meter.Rate5() }

// Rate15 returns the fifteen-minute moving average rate of events per second
// at the time the snapshot was taken.
func (t *TimerSnapshot) Rate15() float64 { return t.meter.Rate15() }

// RateMean returns the meter's mean rate of events per second at the time the
// snapshot was taken.
func (t *TimerSnapshot) RateMean() float64 { return t.meter.RateMean() }

// Snapshot returns the snapshot.
func (t *TimerSnapshot) Snapshot() metrics.Timer { return t }

// StdDev returns the standard deviation of the values at the time the snapshot
// was taken.
func (t *TimerSnapshot) StdDev() float64 { return t.histogram.StdDev() }

// Stop is a no-op.
func (t *TimerSnapshot) Stop() {}

// Sum returns the sum at the time the snapshot was taken.
func (t *TimerSnapshot) Sum() int64 { return t.histogram.Sum() }

// Time panics.
// nolint:iklint
func (*TimerSnapshot) Time(func()) {
	panic("Time called on a TimerSnapshot")
}

// Update panics.
// nolint:iklint
func (*TimerSnapshot) Update(time.Duration) {
	panic("Update called on a TimerSnapshot")
}

// UpdateSince panics.
// nolint:iklint
func (*TimerSnapshot) UpdateSince(time.Time) {
	panic("UpdateSince called on a TimerSnapshot")
}

// Variance returns the variance of the values at the time the snapshot was
// taken.
func (t *TimerSnapshot) Variance() float64 { return t.histogram.Variance() }
