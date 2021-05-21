package circuit

import (
	"sync/atomic"
	"time"
)

type Stats struct {
	Concurrent       int
	SystemLoad       float64
	AverageRT        time.Duration
	ErrorPercent     int
	ErrorSamples     int
	ErrorConsecutive int
}

func (cb *Breaker) GetStats() Stats {
	s := Stats{}
	s.Concurrent = int(atomic.LoadInt64(&cb.callCounts))
	s.SystemLoad, _ = cb.Sys.Load1()
	s.AverageRT = time.Duration(cb.counts.Metric().Mean())
	s.ErrorPercent = int(cb.ErrorRate() * 100)
	s.ErrorConsecutive = int(cb.ConsecFailures())
	s.ErrorSamples = int(cb.Failures() + cb.Successes())
	return s
}
