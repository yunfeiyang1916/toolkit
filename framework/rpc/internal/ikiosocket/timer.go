package ikiosocket

import (
	"sync"
	"time"
)

type Timer struct {
	t    *time.Timer
	cb   func()
	once sync.Once
	s    chan struct{}
}

func NewTimer(d time.Duration, cb func()) *Timer {
	return &Timer{
		t:  time.NewTimer(d),
		cb: cb,
		s:  make(chan struct{}),
	}
}

func (t *Timer) Start() {
	t.once.Do(func() {
		go func() {
			select {
			case <-t.t.C:
				t.cb()
				t.t.Stop()
			case <-t.s:
				return
			}
		}()
	})
}

func (t *Timer) Stop() {
	t.t.Stop()
	close(t.s)
}
