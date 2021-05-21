package timer

import (
	"time"
)

const (
	defaultSlotSize  = 100
	defaultPrecision = time.Millisecond * 500
	defualtBufSize   = 100
)

type Options struct {
	SlotSize    int
	Precision   time.Duration
	BufSize     int
	metricsTags []interface{}
}

type Option func(*Options)

func MetricsTags(tags ...interface{}) Option {
	return func(args *Options) {
		args.metricsTags = tags
	}
}

func SlotSize(size int) Option {
	return func(args *Options) {
		args.SlotSize = size
	}
}

func Precision(p time.Duration) Option {
	return func(args *Options) {
		args.Precision = p
	}
}

func BufSize(size int) Option {
	return func(args *Options) {
		args.BufSize = size
	}
}
