package breaker

import (
	"fmt"
	"sync"
)

type Config struct{}

func NewConfig(configs []BreakerConfig) *Config {
	injectBreakConfig(configs)
	return &Config{}
}

// 每次重新初始化配置
func ReloadConfig(configs []BreakerConfig) {
	ok := isDiffConfig(configs)
	if ok {
		lastVersionConfigs = new(sync.Map)
		initBreakers(configs)
	}
}

func (c *Config) AddConfig(configs []BreakerConfig) {
	injectBreakConfig(configs)
}

func (c *Config) GetBreaker(brkType int, namespace, clientname, resource string) *Breaker {
	brkName := c.getBreakerName(brkType, namespace, clientname, resource)
	watcher := globalWatcher.breakers.Load().(*sync.Map)
	if val, ok := watcher.Load(brkName); ok {
		return val.(*Breaker)
	}

	cfg := c.getConfig(brkType, namespace, clientname)
	if cfg == nil {
		return nil
	}

	brk := initBreaker(BreakerConfig{
		Name:                      brkName,
		ErrorPercentThreshold:     cfg.ErrorPercentThreshold,
		ConsecutiveErrorThreshold: cfg.ConsecutiveErrorThreshold,
		MinSamples:                cfg.MinSamples,
		Break:                     cfg.Break,
	})

	watcher.Store(brkName, brk)
	return brk
}

func (c *Config) getConfig(brkType int, namespace, clientname string) *BreakerConfig {
	if brkType == ClientBreakerType {
		return c.getGlobalClientBreakerConfig(namespace, clientname)
	}
	return c.getGlobalServerBreakerConfig(namespace)
}

/*
namespace@server@* : namespace下server配置生效
namespace@client@* : namespace下client配置生效
namespace@client@service_name@* : namespace、service_name下配置生效

client优先级: namespace@client@service_name@* > namespace@client@*
server优先级: namespace@server@*
*/

func (c *Config) getGlobalServerBreakerConfig(namespace string) *BreakerConfig {
	val, ok := templateConfigs.Load(GetDefaultServerBreakerName(namespace))
	if ok {
		return val.(*BreakerConfig)
	}
	return nil
}

func (c *Config) getGlobalClientBreakerConfig(namespace, clientname string) *BreakerConfig {
	val, ok := templateConfigs.Load(getTemplateClientBreakerName(namespace, clientname))
	if ok {
		return val.(*BreakerConfig)
	}
	val, ok = templateConfigs.Load(GetDefaultClientBreakerName(namespace))
	if ok {
		return val.(*BreakerConfig)
	}
	return nil
}

func (c *Config) getBreakerName(brkType int, namespace, clientname, resource string) string {
	if brkType == ClientBreakerType {
		return GetClientBreakerName(namespace, clientname, resource)
	}
	return GetServerBreakerName(namespace, resource)
}

/*
namespace: 多namespace使用
clientname: 当用于client上时, clientname=app_name+service_name
			当用于server上时, clientname=""
resource: 当用于http服务调用时, resource=uri
          当用于rpc服务调用时, resource=方法签名
*/

// namespace@server@resource
func GetServerBreakerName(namespace, resource string) string {
	return fmt.Sprintf("%s@server@%s", namespace, resource)
}

// namespace@client@clientname@resource
func GetClientBreakerName(namespace, clientname, resource string) string {
	return fmt.Sprintf("%s@client@%s@%s", namespace, clientname, resource)
}

// namespace@server@*
func GetDefaultServerBreakerName(namespace string) string {
	return fmt.Sprintf("%s@server@*", namespace)
}

// namespace@client@*
func GetDefaultClientBreakerName(namespace string) string {
	return fmt.Sprintf("%s@client@*", namespace)
}

// namespace@client@clientname@*
func getTemplateClientBreakerName(namespace, clientname string) string {
	return fmt.Sprintf("%s@client@%s@*", namespace, clientname)
}
