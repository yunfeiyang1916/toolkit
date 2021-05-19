package metrics

import (
	"time"

	metrics "github.com/rcrowley/go-metrics"
)

func GetOrRegisterTimer(name string, r metrics.Registry) metrics.Timer {
	if nil == r {
		r = metrics.DefaultRegistry
	}
	return r.GetOrRegister(name, NewTimer).(metrics.Timer)
}

func Meter(name string, value int, tags ...interface{}) {
	metrics.GetOrRegisterMeter(getMetricName(name, tags), metrics.DefaultRegistry).Mark(int64(value))
}

func Gauge(name string, value int, tags ...interface{}) {
	metrics.GetOrRegisterGauge(getMetricName(name, tags), metrics.DefaultRegistry).Update(int64(value))
}

func Timer(name string, since time.Time, tags ...interface{}) {
	GetOrRegisterTimer(getMetricName(name, tags), metrics.DefaultRegistry).UpdateSince(since)
}

func TimerDuration(name string, duration time.Duration, tags ...interface{}) {
	GetOrRegisterTimer(getMetricName(name, tags), metrics.DefaultRegistry).Update(duration)
}

func CounterInc(name string, tags ...interface{}) {
	metrics.GetOrRegisterCounter(getMetricName(name, tags), metrics.DefaultRegistry).Inc(1)
}

func CounterDec(name string, tags ...interface{}) {
	metrics.GetOrRegisterCounter(getMetricName(name, tags), metrics.DefaultRegistry).Dec(1)
}
