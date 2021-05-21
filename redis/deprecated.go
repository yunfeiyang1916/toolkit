package redis

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
)

// 过时的api， 请不要使用
type RedisPoolConfig struct {
	Addr           string `json:"addr"`
	Password       string `json:"password"`
	MaxIdle        int    `json:"max_idle"`
	IdleTimeout    int    `json:"idle_timeout"`
	MaxActive      int    `json:"max_active"`
	ConnectTimeout int    `json:"connect_timeout"`
	ReadTimeout    int    `json:"read_timeout"`
	WriteTimeout   int    `json:"write_timeout"`
	Database       int    `json:"database"`
}

// 过时的api， 请不要使用
func RedisPoolInit(config *RedisPoolConfig) *redis.Pool {
	opts := []redis.DialOption{}
	if config.ConnectTimeout > 0 {
		opts = append(opts, redis.DialConnectTimeout(time.Duration(config.ConnectTimeout)*time.Millisecond))
	}
	if config.ReadTimeout > 0 {
		opts = append(opts, redis.DialReadTimeout(time.Duration(config.ReadTimeout)*time.Millisecond))
	}
	if config.WriteTimeout > 0 {
		opts = append(opts, redis.DialWriteTimeout(time.Duration(config.WriteTimeout)*time.Millisecond))
	}
	if len(config.Password) > 0 {
		opts = append(opts, redis.DialPassword(config.Password))
	}
	if len(config.Password) > 0 {
		opts = append(opts, redis.DialPassword(config.Password))
	}
	if config.Database != 0 {
		opts = append(opts, redis.DialDatabase(config.Database))
	}
	return redisPoolInit(config.Addr,
		config.Password,
		config.MaxIdle,
		config.IdleTimeout,
		config.MaxActive,
		opts...)
}

func redisPoolInit(server, password string, maxIdle, idleTimeout, maxActive int, options ...redis.DialOption) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     maxIdle,
		IdleTimeout: time.Duration(idleTimeout) * time.Second,
		MaxActive:   maxActive,
		Dial: func() (redis.Conn, error) {
			var c redis.Conn
			var err error
			protocol := "tcp"
			if strings.HasPrefix(server, "unix://") {
				server = strings.TrimLeft(server, "unix://")
				protocol = "unix"
			}
			c, err = redis.Dial(protocol, server, options...)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

// 过时的api， 请不要使用
type Cache struct {
	p *redis.Pool
}

// 过时的api， 请不要使用
func NewCache(config *RedisPoolConfig) *Cache {
	return &Cache{
		p: RedisPoolInit(config),
	}
}

// 过时的api， 请不要使用
func (c *Cache) SetWithExpire(key, value string, expire time.Duration) error {
	client := c.p.Get()
	defer client.Close()
	var err error
	if expire.Seconds() < 1 {
		_, err = client.Do("SET", key, value)
	} else {
		_, err = client.Do("SETEX", key, strconv.Itoa(int(expire.Seconds())), value)

	}
	return err
}

// 过时的api， 请不要使用
func (c *Cache) MultiSetWithExpire(expire time.Duration, data ...string) error {
	if len(data)%2 != 0 {
		return fmt.Errorf("[rediscache] mset error, argument length error")
	}
	setData := make([]interface{}, len(data))
	var keys []interface{}
	for i := 0; i < len(data); {
		keys = append(keys, data[i])
		setData[i] = data[i]
		setData[i+1] = data[i+1]
		i = i + 2
	}
	client := c.p.Get()
	defer client.Close()
	_, err := client.Do("MSET", setData...)
	if err != nil {
		return err
	}
	expireInt := int(expire.Seconds())
	if expire > 1 {
		for _, k := range keys {
			client.Do("EXPIRE", k, expireInt)
		}
	}
	return err
}

// 过时的api， 请不要使用
func (c *Cache) Get(key string) ([]byte, error) {
	client := c.p.Get()
	rsp, err := client.Do("GET", key)
	client.Close()
	if err != nil {
		return nil, err
	}
	if rsp == nil {
		return nil, nil
	}
	data, err := redis.Bytes(rsp, err)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// 过时的api， 请不要使用
func (c *Cache) MultiGet(ks ...string) ([][]byte, error) {
	var keys []interface{}
	for _, k := range ks {
		keys = append(keys, k)
	}
	client := c.p.Get()
	rsp, err := redis.ByteSlices(client.Do("MGET", keys...))
	client.Close()
	return rsp, err
}
