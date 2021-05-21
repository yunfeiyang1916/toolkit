package ikiosocket

import (
	"time"
)

type Options struct {
	RequestTimeout time.Duration
}

func Timeout(t time.Duration) Option {
	return func(opt *Options) {
		opt.RequestTimeout = t
	}
}
