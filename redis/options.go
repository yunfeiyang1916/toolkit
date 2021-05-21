package redis

import (
	"fmt"
)

// RedisConfig结构体作为NewRedis函数的参数，用于初始化Redis结构体。
type RedisConfig struct {
	// 这个字段会上报到stats和trace系统
	ServerName string `json:"server_name"`

	// Redis服务器的host和port "localhost:6379"
	Addr string `json:"addr"`

	// 连接到redis服务器的密码
	Password string `json:"password"`

	// 在连接池中可以存在的最大空闲连接数
	MaxIdle int `json:"max_idle"`

	// Maximum number of connections allocated by the pool at a given time.
	// When zero, there is no limit on the number of connections in the pool.
	MaxActive int `json:"max_active"`

	// Close connections after remaining idle for this duration. If the value
	// is zero, then idle connections are not closed. Applications should set
	// the timeout to a value less than the server's timeout.
	IdleTimeout int `json:"idle_timeout"`

	// Specifies the timeout for connecting to the Redis server.
	ConnectTimeout int `json:"connect_timeout"`
	ReadTimeout    int `json:"read_timeout"`

	// WriteTimeout specifies the timeout for writing a single command.
	WriteTimeout int `json:"write_timeout"`

	// Database specifies the database to select when dialing a connection.
	Database int `json:"database"`

	// 慢日志打印， 单位毫秒, 如果一个请求的时间慢于这个值，
	// 将会把详细信息打印在slow.log日志中
	SlowTime int `json:"slow_time"`

	// 内部重试次数
	Retry int `json:"retry"`
}

func (o *RedisConfig) init() error {
	if o.ServerName == "" {
		return fmt.Errorf("redis: ServerName not allowed empty string")
	}
	if o.Addr == "" {
		return fmt.Errorf("redis: Addr not allowed empty string")
	}
	if o.Database < 0 {
		return fmt.Errorf("redis: Database less than zero")
	}

	if o.MaxIdle < 0 {
		o.MaxIdle = 100
	}
	if o.MaxActive < 0 {
		o.MaxActive = 100
	}
	if o.IdleTimeout < 0 {
		o.IdleTimeout = 100
	}
	if o.ReadTimeout < 0 {
		o.ReadTimeout = 50
	}
	if o.WriteTimeout < 0 {
		o.WriteTimeout = 50
	}
	if o.ConnectTimeout < 0 {
		o.ConnectTimeout = 300
	}

	if o.SlowTime <= 0 {
		o.SlowTime = 100
	}

	if o.Retry < 0 {
		o.Retry = 0
	}

	return nil
}
