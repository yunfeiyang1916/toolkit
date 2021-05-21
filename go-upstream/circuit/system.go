// +build !windows

package circuit

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	load "github.com/shirou/gopsutil/load"
)

var errSystem = errors.New("defaultSystem: not init")

type System interface {
	Load1() (float64, error)
}

var once sync.Once
var stat atomic.Value

type defaultSystem struct{}

func (defaultSystem) Load1() (float64, error) {
	once.Do(func() {
		avg, err := load.AvgWithContext(context.Background())
		if err == nil {
			stat.Store(avg)
		}
		go func() {
			for {
				time.Sleep(time.Second * 3)
				avg, err := load.AvgWithContext(context.Background())
				if err != nil {
					continue
				}
				stat.Store(avg)
			}
		}()
	})
	if v := stat.Load(); v == nil {
		return 0.0, errSystem
	} else {
		return v.(*load.AvgStat).Load1, nil
	}
}

type mockSystem struct {
	open  int32
	value float64
	def   defaultSystem
}

func (d *mockSystem) Open() {
	atomic.StoreInt32(&d.open, 1)
}

func (d *mockSystem) Close() {
	atomic.StoreInt32(&d.open, 0)
}

func (d *mockSystem) Load1() (float64, error) {
	if atomic.LoadInt32(&d.open) == 1 {
		return d.value, nil
	}
	return d.def.Load1()
}
