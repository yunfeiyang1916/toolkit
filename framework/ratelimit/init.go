package ratelimit

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/yunfeiyang1916/toolkit/logging"
)

var globalWatcher watcher

type watcher struct {
	limiters atomic.Value
}

type LimiterConfig struct {
	Name   string `toml:"name"`
	Limits int    `toml:"limits"`
	Open   bool   `toml:"open"`
}

func init() {
	globalWatcher.limiters.Store(&sync.Map{})
}

// 有配置变更, 每次构造新的limiter
func initLimiter(c LimiterConfig) *Limiter {
	var lim *Limiter
	if c.Open {
		lim = NewLimiter(Limit(c.Limits), c.Limits)
	} else {
		lim = NewLimiter(Inf, c.Limits)
	}
	logging.GenLogf("on ratelimit, name:%s, open:%t, limits:%d", c.Name, c.Open, c.Limits)
	return lim
}

func initLimiters(configs []LimiterConfig) {
	if len(configs) == 0 {
		return
	}
	limiters := &sync.Map{}
	for _, v := range configs {
		lastVersionConfigs.Store(v.Name, v)

		if strings.Contains(v.Name, "*") {
			templateConfigs.Store(v.Name, &LimiterConfig{
				Name:   v.Name,
				Limits: v.Limits,
				Open:   v.Open,
			})
			logging.GenLogf("on ratelimit, template name:%s, open:%t, limits:%d", v.Name, v.Open, v.Limits)
			continue
		}

		limiters.Store(v.Name, initLimiter(v))
	}
	globalWatcher.limiters.Store(limiters)
}

func injectLimiterConfig(configs []LimiterConfig) {
	if len(configs) == 0 {
		return
	}
	watcher := globalWatcher.limiters.Load().(*sync.Map)
	for _, v := range configs {
		lastVersionConfigs.Store(v.Name, v)

		if strings.Contains(v.Name, "*") {
			if _, ok := templateConfigs.Load(v.Name); !ok {
				templateConfigs.Store(v.Name, &LimiterConfig{
					Name:   v.Name,
					Limits: v.Limits,
					Open:   v.Open,
				})
			}
			continue
		}

		_, ok := watcher.Load(v.Name)
		if !ok {
			watcher.Store(v.Name, initLimiter(v))
		}
	}
}

func InitLimiters(configs []LimiterConfig) {
	initLimiters(configs)
}

func Allow(name string) bool {
	watcher := globalWatcher.limiters.Load().(*sync.Map)
	if val, ok := watcher.Load(name); ok {
		lim := val.(*Limiter)
		return lim.Allow()
	}
	return true
}
