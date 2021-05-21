package breaker

import (
	"sync"
)

var circuitSetting = new(sync.Map) // string -> Setting breaker kv结构

// name: 完整的breaker name; 包含全局配置
func Configure(name string, config *BreakerConfig) {
	circuitSetting.Store(name, config)
}

func getSetting(name string) *BreakerConfig {
	setting, ok := circuitSetting.Load(name)
	if !ok {
		return nil
	}
	return setting.(*BreakerConfig)
}
