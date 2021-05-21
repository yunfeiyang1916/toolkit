package log

import "github.com/yunfeiyang1916/toolkit/logging"

type Kit interface {
	// Business log
	B() *logging.Logger
	// Access log
	A() *logging.Logger
	// Error log
	E() *logging.Logger
}

type kit struct {
	b, a, e *logging.Logger
}

func NewKit(b, a, e *logging.Logger) Kit {
	return kit{
		b: b,
		a: a,
		e: e,
	}
}

func (c kit) B() *logging.Logger {
	return c.b
}

func (c kit) A() *logging.Logger {
	return c.a
}

func (c kit) E() *logging.Logger {
	return c.e
}
