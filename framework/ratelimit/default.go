package ratelimit

import (
	"reflect"
	"sync"

	dutils "github.com/yunfeiyang1916/toolkit/framework/utils"
)

var (
	templateConfigs    = new(sync.Map) // 包含有通配符的配置 key: limName, val: *LimiterConfig
	lastVersionConfigs = new(sync.Map) // map[string]LimiterConfig
)

func InitDefaultConfig(configs []LimiterConfig) {
	for _, v := range configs {
		if !v.Open && v.Limits == 0 {
			v.Open = true
			v.Limits = 10000
		}
		c := LimiterConfig{
			Name:   v.Name,
			Limits: v.Limits,
			Open:   v.Open,
		}
		templateConfigs.Store(v.Name, &c)

		lastVersionConfigs.Store(v.Name, c)
	}
}

func isDiffConfig(configs []LimiterConfig) bool {
	if len(configs) != dutils.LenSyncMap(lastVersionConfigs) {
		return true
	}

	for _, v := range configs {
		val, ok := lastVersionConfigs.Load(v.Name)
		if !ok {
			return true
		}
		old := val.(LimiterConfig)
		if !reflect.DeepEqual(v, old) {
			return true
		}
	}
	return false
}
