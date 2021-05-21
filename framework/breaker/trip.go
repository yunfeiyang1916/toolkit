package breaker

// ConsecutiveTripFunc returns a TripFunc that trips whenever
// the consecutive failure count meets the threshold.
func ConsecutiveTripFunc(name string) TripFunc {
	return func(cb *Breaker) error {
		setting := getSetting(name)
		if setting != nil {
			if setting.ConsecutiveErrorThreshold > 0 && int(cb.ConsecFailures()) >= setting.ConsecutiveErrorThreshold {
				return ErrConsecutiveThreshold
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
func ErrorPercentTripFunc(name string) TripFunc {
	return func(cb *Breaker) error {
		samples := cb.Failures() + cb.Successes()
		setting := getSetting(name)
		if setting != nil {
			if setting.ErrorPercentThreshold > 0 && setting.MinSamples > 0 && int(samples) >= setting.MinSamples && int(cb.ErrorRate()*100) >= setting.ErrorPercentThreshold {
				return ErrPercentThreshold
			}
		}
		return nil
	}
}

/*
func ThresholdTripFunc(name string) TripFunc {
	return func(cb *Breaker) error {
		setting := getSetting(name)
		if setting != nil && setting.Open {
			if int(cb.Failures()) >= setting.Threshold.Threshold {
				return errors.New("breaker: error threshold")
			}
		}
		return nil
	}
}
*/
