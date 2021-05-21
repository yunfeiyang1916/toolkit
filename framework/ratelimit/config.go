package ratelimit

import (
	"fmt"
	"sync"
)

type Config struct{}

func NewConfig(configs []LimiterConfig) *Config {
	injectLimiterConfig(configs)
	return &Config{}
}

func ReloadConfig(configs []LimiterConfig) {
	ok := isDiffConfig(configs)
	if ok {
		lastVersionConfigs = new(sync.Map)
		initLimiters(configs)
	}
}

func (c *Config) AddConfig(configs []LimiterConfig) {
	injectLimiterConfig(configs)
}

func (c *Config) GetLimiter(limType int, namespace, clientname, resource string) *Limiter {
	limName := c.getLimiterName(limType, namespace, clientname, resource)
	watcher := globalWatcher.limiters.Load().(*sync.Map)
	if val, ok := watcher.Load(limName); ok {
		return val.(*Limiter)
	}

	cfg := c.getConfig(limType, namespace, clientname, resource)
	if cfg == nil {
		return nil
	}

	lim := initLimiter(LimiterConfig{
		Name:   limName,
		Limits: cfg.Limits,
		Open:   cfg.Open,
	})

	watcher.Store(limName, lim)
	return lim
}

func (c *Config) getConfig(limType int, namespace, clientname, resource string) *LimiterConfig {
	if limType == ClientLimiterType {
		return c.getClientGlobalConfig(namespace, clientname)
	}
	return c.getServerGlobalConfig(namespace, resource)
}

/*
namespace@server@* : namespace下server配置生效
namespace@client@* : namespace下client配置生效
namespace@client@service_name@* : namespace、service_name下配置生效
namespace@server@*@resource : namespace、resource下配置生效
[server.limiter]
"*@/api/sms/send"={limits=100, open=true}
"user.account.login@*"={limits=10, open=true} 不支持通配

client优先级: namespace@client@service_name@* > namespace@client@*
server优先级: namespace@server@*@resource > namespace@server@*
*/
func (c *Config) getServerGlobalConfig(namespace, resource string) *LimiterConfig {
	val, ok := templateConfigs.Load(getTemplateServerLimiterName(namespace, resource))
	if ok {
		return val.(*LimiterConfig)
	}
	val, ok = templateConfigs.Load(GetDefaultServerLimiterName(namespace))
	if ok {
		return val.(*LimiterConfig)
	}
	return nil
}

func (c *Config) getClientGlobalConfig(namespace, clientname string) *LimiterConfig {
	val, ok := templateConfigs.Load(getTemplateClientLimiterName(namespace, clientname))
	if ok {
		return val.(*LimiterConfig)
	}
	val, ok = templateConfigs.Load(GetDefaultClientLimiterName(namespace))
	if ok {
		return val.(*LimiterConfig)
	}
	return nil
}

func (c *Config) getLimiterName(limType int, namespace, clientname, resource string) string {
	if limType == ClientLimiterType {
		return GetClientLimiterName(namespace, clientname, resource)
	}
	return GetServerLimiterName(namespace, clientname, resource)
}

/*
namespace: 多namespace使用
clientname: 当用于client上时, clientname=app_name+service_name(下游服务)
			当用于server上时, clientname=app_name+service_name(上游服务)
resource: 当用于http服务调用时, resource=uri
          当用于rpc服务调用时, resource=方法签名
*/
func GetServerLimiterName(namespace, clientname, resource string) string {
	return fmt.Sprintf("%s@server@%s@%s", namespace, clientname, resource)
}

func GetClientLimiterName(namespace, clientname, resource string) string {
	return fmt.Sprintf("%s@client@%s@%s", namespace, clientname, resource)
}

func GetDefaultServerLimiterName(namespace string) string {
	return fmt.Sprintf("%s@server@*", namespace)
}

func GetDefaultClientLimiterName(namespace string) string {
	return fmt.Sprintf("%s@client@*", namespace)
}

func getTemplateServerLimiterName(namespace, resource string) string {
	return fmt.Sprintf("%s@server@*@%s", namespace, resource)
}

func getTemplateClientLimiterName(namespace, clientname string) string {
	return fmt.Sprintf("%s@client@%s@*", namespace, clientname)
}
