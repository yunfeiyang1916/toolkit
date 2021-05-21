package circuit

type BreakerError interface {
	Error() string
	Breaker() bool
}

type breakerError struct {
	message string
}

func NewBreakerError(m string) BreakerError {
	return breakerError{m}
}

func (breakerError) Breaker() bool {
	return true
}

func (b breakerError) Error() string {
	return b.message
}
