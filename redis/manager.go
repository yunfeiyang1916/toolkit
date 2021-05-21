package redis

import (
	"fmt"
	"sync"
)

// Manager 用于管理多个redis client， 通常在rpc-go基础库中使用
type Manager struct {
	redisMap map[string]*Redis
	sync.RWMutex
}

// NewManager 初始化一个Manager
func NewManager(c []RedisConfig) (*Manager, error) {
	m := &Manager{
		redisMap: map[string]*Redis{},
	}
	for _, config := range c {
		r, err := NewRedis(&config)
		if err == nil {
			m.redisMap[config.ServerName] = r
		} else {
			return nil, fmt.Errorf("redis: init redis: %s error: %s", config.ServerName, err)
		}
	}
	return m, nil
}

func (m *Manager) Add(name string, r *Redis) {
	m.Lock()
	defer m.Unlock()
	m.redisMap[name] = r
}

// Get 使用RedisConfig结构体中的ServerName字段作为key来获取redis client
func (m *Manager) Get(name string) *Redis {
	m.RLock()
	defer m.RUnlock()
	return m.redisMap[name]
}
