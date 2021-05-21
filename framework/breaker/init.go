package breaker

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/yunfeiyang1916/toolkit/logging"
)

type watcher struct {
	breakers atomic.Value
}

var globalWatcher watcher

type BreakerConfig struct {
	Name                      string `toml:"name"`
	ErrorPercentThreshold     int    `toml:"error_percent_threshold"`
	ConsecutiveErrorThreshold int    `toml:"consecutive_error_threshold"`
	MinSamples                int    `toml:"minsamples"`
	Break                     bool   `toml:"break"`
}

func init() {
	globalWatcher.breakers.Store(&sync.Map{})
}

// 有配置变更, 每次构造新的breaker
func initBreaker(c BreakerConfig) *Breaker {
	Configure(c.Name, &c)

	brk := NewBreakerWithOptions(&Options{
		Name: c.Name,
	})
	if c.Break {
		brk.Break()
	}

	logging.GenLogf("on breaker, name:%s, break:%v, error_percent:%d, consecutive_error:%d, minsamples:%d", c.Name, c.Break, c.ErrorPercentThreshold, c.ConsecutiveErrorThreshold, c.MinSamples)
	return brk
}

// 重新加载配置使用
func initBreakers(configs []BreakerConfig) {
	if len(configs) == 0 {
		return
	}
	breakers := &sync.Map{}
	for _, v := range configs {
		lastVersionConfigs.Store(v.Name, v)

		if strings.Contains(v.Name, "*") {
			templateConfigs.Store(v.Name, &BreakerConfig{
				Name:                      v.Name,
				ErrorPercentThreshold:     v.ErrorPercentThreshold,
				ConsecutiveErrorThreshold: v.ConsecutiveErrorThreshold,
				MinSamples:                v.MinSamples,
				Break:                     v.Break,
			})
			logging.GenLogf("on breaker, template name:%s, break:%v, error_percent:%d, consecutive_error:%d, minsamples:%d", v.Name, v.Break, v.ErrorPercentThreshold, v.ConsecutiveErrorThreshold, v.MinSamples)
			continue
		}

		breakers.Store(v.Name, initBreaker(v))
	}
	globalWatcher.breakers.Store(breakers)
}

// 初始化时使用
func injectBreakConfig(configs []BreakerConfig) {
	if len(configs) == 0 {
		return
	}
	watcher := globalWatcher.breakers.Load().(*sync.Map)
	for _, v := range configs {
		lastVersionConfigs.Store(v.Name, v)

		if strings.Contains(v.Name, "*") {
			if _, ok := templateConfigs.Load(v.Name); !ok {
				templateConfigs.Store(v.Name, &BreakerConfig{
					Name:                      v.Name,
					ErrorPercentThreshold:     v.ErrorPercentThreshold,
					ConsecutiveErrorThreshold: v.ConsecutiveErrorThreshold,
					MinSamples:                v.MinSamples,
					Break:                     v.Break,
				})
				continue
			}
		}

		if _, ok := watcher.Load(v.Name); !ok {
			watcher.Store(v.Name, initBreaker(v))
		}
	}
}

func InitBreakers(configs []BreakerConfig) {
	initBreakers(configs)
}

func GetBreaker(name string) *Breaker {
	watcher := globalWatcher.breakers.Load().(*sync.Map)
	if val, ok := watcher.Load(name); ok {
		return val.(*Breaker)
	}
	return nil
}
