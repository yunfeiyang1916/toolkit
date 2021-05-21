package breaker

import (
	"reflect"
	"sync"

	dutils "github.com/yunfeiyang1916/toolkit/framework/utils"
)

var (
	templateConfigs    = new(sync.Map) // 包含有通配符的配置 key: brkName, val: *BreakerConfig
	lastVersionConfigs = new(sync.Map) // map[string]BreakerConfig
)

func InitDefaultConfig(configs []BreakerConfig) {
	for _, v := range configs {
		if v.ErrorPercentThreshold == 0 && v.ConsecutiveErrorThreshold == 0 && v.MinSamples == 0 && !v.Break {
			v.ErrorPercentThreshold = 80
			v.ConsecutiveErrorThreshold = 200
			v.MinSamples = 1000
			v.Break = false
		}
		c := BreakerConfig{
			Name:                      v.Name,
			ErrorPercentThreshold:     v.ErrorPercentThreshold,
			ConsecutiveErrorThreshold: v.ConsecutiveErrorThreshold,
			MinSamples:                v.MinSamples,
			Break:                     v.Break,
		}
		templateConfigs.Store(v.Name, &c)

		lastVersionConfigs.Store(v.Name, c)
	}
}

func isDiffConfig(configs []BreakerConfig) bool {
	if len(configs) != dutils.LenSyncMap(lastVersionConfigs) {
		return true
	}

	for _, v := range configs {
		val, ok := lastVersionConfigs.Load(v.Name)
		if !ok {
			return true
		}
		old := val.(BreakerConfig)
		if !reflect.DeepEqual(v, old) {
			return true
		}
	}
	return false
}
