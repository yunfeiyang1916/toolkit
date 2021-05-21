package circuit

import (
	"time"

	metrics "github.com/rcrowley/go-metrics"
)

// metric info in windows.
type metric struct {
	sample metrics.Sample
}

func newMetric() *metric {
	return &metric{metrics.NewUniformSample(200)}
}

func (m *metric) Reset() {
	m.sample.Clear()
}

// Mean return average run duration.
func (m *metric) Mean() float64 {
	return m.sample.Mean()
}

// Count return sample counts.
func (m *metric) Count() int64 {
	return m.sample.Count()
}

func (m *metric) Update(r metricResult) {
	m.sample.Update(int64(r.RunDuration))
}

type metricResult struct {
	RunDuration time.Duration
}
