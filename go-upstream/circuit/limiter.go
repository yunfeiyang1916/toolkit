// This file implement some rate limiter.
package circuit

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/paulbellamy/ratecounter"
	"go.uber.org/ratelimit"
)

const (
	QPSReject      = "reject"
	QPSLeakyBucket = "leaky_bucket"
	QPSSlowStart   = "slow_start"
)

// SystemLoadCheckerFunc return a CheckerFunc that return error whenever the
// system load1 exceeds the threshold.
func SystemLoadCheckerFunc(name string) CheckerFunc {
	return func(cb *Breaker) error {
		setting := getSetting(name)
		if setting != nil && setting.SystemLoads.Open {
			val, err := cb.Sys.Load1()
			if err != nil {
				return nil
			}
			if val > setting.SystemLoads.Threshold {
				return ErrSystemLoad
			}
		}
		return nil
	}
}

// MaxConcurrentCheckerFunc return a CheckerFunc that return error whenever the
// concurrent request number exceed the threshold.
func MaxConcurrentCheckerFunc(name string) CheckerFunc {
	return func(cb *Breaker) error {
		setting := getSetting(name)
		if setting != nil && setting.MaxConcurrent.Open {
			if atomic.LoadInt64(&cb.callCounts) > setting.MaxConcurrent.Threshold {
				return ErrMaxConcurrent
			}
		}
		return nil
	}
}

// QPSCheckerFunc return a CheckerFunc that return error whenever request rate
// exceeds the QPS limter currently used.
func QPSCheckerFunc(name string) CheckerFunc {
	checkers := map[string]CheckerFunc{
		QPSReject:      QPSRejectCheckerFunc(name),
		QPSLeakyBucket: QPSLeakyBucketCheckerFunc(name),
		QPSSlowStart:   QPSSlowStartCheckerFunc(name),
	}
	return func(cb *Breaker) error {
		setting := getSetting(name)
		if setting != nil && setting.QPSLimit.Open {
			if call, ok := checkers[setting.QPSLimit.Strategy]; ok {
				return call(cb)
			}
		}
		return nil
	}
}

// QPSRejectCheckerFunc return a CheckerFunc that return error whenever the
// request number per second exceed the threshold.
func QPSRejectCheckerFunc(name string) CheckerFunc {
	//counter := ratecounter.NewAvgRateCounter(1 * time.Second)
	counter := ratecounter.NewRateCounter(1 * time.Second)
	return func(cb *Breaker) error {
		setting := getSetting(name)
		if setting == nil || !setting.QPSLimit.Open {
			return nil
		}
		if int64(counter.Rate()) > setting.QPSLimit.Threshold {
			return ErrRateLimit
		}
		counter.Incr(1)
		return nil
	}
}

// QPSSlowStartCheckerFunc return a CheckerFunc that return error whenever
// the rate of incoming request exceeds the QPSSlowStartCheckerFunc's growing
// rate.
func QPSSlowStartCheckerFunc(name string) CheckerFunc {
	return func(cb *Breaker) error {
		// TODO
		return nil
	}
}

// QPSLeakyBucketCheckerFunc return a CheckerFunc that will block the function
// whenever the request QPS exceeds the limit using leaky bucket.
// QPSLeakyBucketCheckerFunc must be checked after the MaxConcurrentCheckerFunc.
func QPSLeakyBucketCheckerFunc(name string) CheckerFunc {
	var lock sync.Mutex
	type wrap struct {
		lr    ratelimit.Limiter
		limit int64
	}
	rls := new(sync.Map)
	return func(cb *Breaker) error {
		setting := getSetting(name)
		if setting == nil || !setting.QPSLimit.Open {
			return nil
		}
		limit := setting.QPSLimit.Threshold

		actual, _ := rls.LoadOrStore(name, wrap{})
		if w, _ := actual.(wrap); w.limit != limit {
			lock.Lock()
			defer lock.Unlock()

			if actual, _ := rls.Load(name); actual.(wrap).limit != limit {
				rls.Store(name, wrap{
					lr:    ratelimit.New(int(limit)),
					limit: limit,
				})
			}
		}

		actual, _ = rls.Load(name)
		actual.(wrap).lr.Take()
		return nil
	}
}
