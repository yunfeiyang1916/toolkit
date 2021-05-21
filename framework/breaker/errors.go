package breaker

type Error interface {
	Error() string
}

type breakerError struct {
	message string
}

func NewError(m string) error {
	return breakerError{m}
}

func (b breakerError) Error() string {
	return b.message
}
