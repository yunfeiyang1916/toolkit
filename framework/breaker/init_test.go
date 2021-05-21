package breaker

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
)

func TestInitBreakers(t *testing.T) {
	c := []BreakerConfig{
		{
			Name:                      "@inf.secret.base@/api/sms/send",
			ErrorPercentThreshold:     50,
			ConsecutiveErrorThreshold: 10,
			MinSamples:                100,
			Break:                     true,
		},
		{
			Name:                      "@inf.secret.base@/api/sms/check",
			ErrorPercentThreshold:     50,
			ConsecutiveErrorThreshold: 10,
			MinSamples:                100,
			Break:                     true,
		},
		{
			Name:                      "@inf.secret.base@/api/sms/receipt",
			ErrorPercentThreshold:     50,
			ConsecutiveErrorThreshold: 10,
			MinSamples:                100,
			Break:                     false,
		},
	}
	InitBreakers(c)

	watcher := globalWatcher.breakers.Load().(*sync.Map)

	watcher.Range(func(k, v interface{}) bool {
		fmt.Printf("key:%+v\n", k)
		return true
	})

	_, ok := watcher.Load("@inf.secret.base@/api/sms/send")
	assert.Equal(t, ok, true)

	_, ok = watcher.Load("@inf.secret.base@/api/sms/check")
	assert.Equal(t, ok, true)

	_, ok = watcher.Load("@inf.secret.base@/api/sms/receipt")
	assert.Equal(t, ok, true)
}

func TestConsecutiveErrorThreshold(t *testing.T) {
	c := []BreakerConfig{
		{
			Name:                      "@inf.secret.base@/api/sms/send",
			ErrorPercentThreshold:     0,
			ConsecutiveErrorThreshold: 2,
			MinSamples:                10,
			Break:                     false,
		},
	}
	InitBreakers(c)

	brk := GetBreaker("@inf.secret.base@/api/sms/send")
	assert.Equal(t, brk != nil, true)

	for i := 0; i < 20; i++ {
		if err := brk.Call(func() error {
			if i > 5 && i < 8 {
				return errors.New("bad request")
			}
			return nil
		}, 0); err != nil {
			if i < 8 {
				assert.Equal(t, err.Error(), "bad request")
			} else {
				assert.Equal(t, err.Error(), ErrConsecutiveThreshold.Error())
			}
		}
	}
}

func TestErrorPercentThreshold(t *testing.T) {
	c := []BreakerConfig{
		{
			Name:                      "@inf.secret.base@/api/sms/send",
			ErrorPercentThreshold:     20,
			ConsecutiveErrorThreshold: 0,
			MinSamples:                10,
			Break:                     false,
		},
	}
	InitBreakers(c)

	brk := GetBreaker("@inf.secret.base@/api/sms/send")
	assert.Equal(t, brk != nil, true)

	for i := 0; i < 40; i++ {
		if err := brk.Call(func() error {
			if i%5 == 0 || i%5 == 2 {
				return errors.New("bad request")
			}
			return nil
		}, 0); err != nil {
			if i < 11 {
				assert.Equal(t, err.Error(), "bad request")
			} else {
				assert.Equal(t, err.Error(), ErrPercentThreshold.Error())
			}
		}
	}
}

func TestBreakError(t *testing.T) {
	c := []BreakerConfig{
		{
			Name:                      "@inf.secret.base@/api/sms/send",
			ErrorPercentThreshold:     20,
			ConsecutiveErrorThreshold: 0,
			MinSamples:                10,
			Break:                     true,
		},
	}
	InitBreakers(c)

	brk := GetBreaker("@inf.secret.base@/api/sms/send")
	assert.Equal(t, brk != nil, true)

	for i := 0; i < 40; i++ {
		if err := brk.Call(func() error {
			if i%5 == 0 || i%5 == 2 {
				return errors.New("bad request")
			}
			return nil
		}, 0); err != nil {
			assert.Equal(t, err.Error(), ErrOpen.Error())
		}
	}
}

func Test_atomic_value(t *testing.T) {
	type Model struct {
		values atomic.Value
	}

	var global Model

	global.values.Store(&sync.Map{})

	fmt.Println("================ init values test ==============")
	vals := &sync.Map{}

	vals.Store("k1", "v1")
	vals.Store("k2", "v2")
	vals.Store("k3", "v3")

	global.values.Store(vals)

	fmt.Println("================ update values test ==============")
	values := global.values.Load().(*sync.Map)
	v, _ := values.Load("k1")
	assert.Equal(t, "v1", v.(string))
	values.Store("k1", "vv1")

	values2 := global.values.Load().(*sync.Map)
	vv, _ := values2.Load("k1")
	assert.Equal(t, "vv1", vv.(string))

	fmt.Println("================ add values test ==============")
	vals3 := &sync.Map{}

	vals3.Store("kk1", "v1")
	vals3.Store("kk2", "v2")
	vals3.Store("kk3", "v3")

	global.values.Store(vals3)

	values3 := global.values.Load().(*sync.Map)
	vvv, _ := values3.Load("k1")
	assert.Equal(t, nil, vvv)

	vvv2, _ := values3.Load("kk1")
	assert.Equal(t, "v1", vvv2.(string))
}

func Test_atomic_value2(t *testing.T) {
	type Model struct {
		values atomic.Value
	}

	var global Model

	global.values.Store(&sync.Map{})

	go func() {
		values := global.values.Load().(*sync.Map)
		for i := 0; i < 20; i++ {
			if _, ok := values.Load(fmt.Sprintf("k%d", i)); !ok {
				values.Store(fmt.Sprintf("k%d", i), fmt.Sprintf("v%d", i))
			}
		}
	}()

	go func() {
		values := global.values.Load().(*sync.Map)
		for i := 0; i < 20; i++ {
			if _, ok := values.Load(fmt.Sprintf("kk%d", i)); !ok {
				values.Store(fmt.Sprintf("kk%d", i), fmt.Sprintf("vv%d", i))
			}
		}
	}()

	//go func() {
	//	values := global.values.Load().(*sync.Map)
	//	for i := 0; i < 20; i++ {
	//		if _, ok := values.Load(fmt.Sprintf("k%d", i)); !ok {
	//			values.Store(fmt.Sprintf("k%d", i), fmt.Sprintf("vvv%d", i))
	//		}
	//	}
	//}()

	time.Sleep(5 * time.Millisecond)

	values := global.values.Load().(*sync.Map)
	values.Range(func(key, value interface{}) bool {
		fmt.Printf("key:%s, val:%s\n", key.(string), value.(string))
		return true
	})
}
