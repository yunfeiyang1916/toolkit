package ratelimit

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// golang.org/x/time/rate
// 采用令牌桶的算法实现

func TestInitLimiters(t *testing.T) {
	c := []LimiterConfig{
		{Name: "@inf.sms.base@/api/sms/send", Limits: 100, Open: false},
		{Name: "@inf.sms.base@/api/sms/check", Limits: 100, Open: false},
		{Name: "@inf.sms.base@/api/sms/receipt", Limits: 100, Open: false},
	}
	InitLimiters(c)
	watcher := globalWatcher.limiters.Load().(*sync.Map)
	_, ok := watcher.Load("@inf.sms.base@/api/sms/send")
	assert.Equal(t, true, ok)
	_, ok = watcher.Load("@inf.sms.base@/api/sms/check")
	assert.Equal(t, true, ok)
	_, ok = watcher.Load("@inf.sms.base@/api/sms/receipt")
	assert.Equal(t, true, ok)
}

func TestAllow(t *testing.T) {
	c := []LimiterConfig{
		{Name: "@inf.sms.base@/api/sms/send", Limits: 5, Open: true},
	}
	InitLimiters(c)

	for i := 0; i < 10; i++ {
		v := Allow("@inf.sms.base@/api/sms/send")
		if i >= 5 {
			assert.Equal(t, false, v)
		} else {
			assert.Equal(t, true, v)
		}

		v = Allow("unknown")
		assert.Equal(t, true, v)
	}
}

func TestAllowAll(t *testing.T) {
	c := []LimiterConfig{
		{Name: "@inf.sms.base@/api/sms/send", Limits: 5, Open: false},
	}
	InitLimiters(c)

	for i := 0; i < 10; i++ {
		v := Allow("@inf.sms.base@/api/sms/send")
		assert.Equal(t, true, v)

		v = Allow("unknown")
		assert.Equal(t, true, v)
	}
}
