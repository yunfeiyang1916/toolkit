package circuit

import (
	"time"
)

// TripFunc is a function called by a Breaker's Fail() function and determines whether
// the breaker should trip. It will receive the Breaker as an argument and returns a
// boolean. By default, a Breaker has no TripFunc.
type TripFunc func(*Breaker) bool

// AverageRTTripFunc returns a TripFunc with that trips whenever
// the success response time excced the threshold.
func AverageRTTripFunc(name string) CheckerFunc {
	return func(cb *Breaker) error {
		setting := getSetting(name)
		if setting != nil && setting.AverageRT.Open {
			mean := cb.counts.Metric().Mean()
			if time.Duration(mean) > setting.AverageRT.RT {
				return ErrAverageRT
			}
		}
		return nil
	}
}

// ConsecutiveTripFunc returns a TripFunc that trips whenever
// the consecutive failure count meets the threshold.
func ConsecutiveTripFunc(name string) CheckerFunc {
	return func(cb *Breaker) error {
		setting := getSetting(name)
		if setting != nil && setting.ConsecutiveError.Open {
			if cb.ConsecFailures() >= setting.ConsecutiveError.Threshold {
				return ErrConsecutive
			}
		}
		return nil
	}
}

// RateTripFunc returns a TripFunc that trips whenever the
// error rate hits the threshold. The error rate is calculated as such:
// f = number of failures
// s = number of successes
// e = f / (f + s)
// The error rate is calculated over a sliding window of 10 seconds (by default)
// This TripFunc will not trip until there have been at least minSamples events.
func ErrorPercentTripFunc(name string) CheckerFunc {
	return func(cb *Breaker) error {
		samples := cb.Failures() + cb.Successes()
		setting := getSetting(name)
		if setting != nil && setting.ErrorPercent.Open {
			if samples >= setting.ErrorPercent.MinSamples && int64(cb.ErrorRate()*100) >= setting.ErrorPercent.Threshold {
				return ErrPercent
			}
		}
		return nil
	}
}
