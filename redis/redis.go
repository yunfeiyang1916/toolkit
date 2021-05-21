package redis

import (
	"io"

	"github.com/yunfeiyang1916/toolkit/go-tls"

	"github.com/garyburd/redigo/redis"
	"github.com/olekukonko/tablewriter"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	log "github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/utils"

	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var serverLocalPid = os.Getpid()

var logFormat = "2006/01/02 15:04:05"

type Redis struct {
	*ctxRedis
}

type ctxRedis struct {
	pool     *redis.Pool
	opts     *RedisConfig
	lastTime int64

	ctx context.Context
}

var maps sync.Map

func init() {
	go func() {
		for {
			time.Sleep(time.Second * 120)
			b := &bytes.Buffer{}
			table := tablewriter.NewWriter(b)
			table.SetHeader([]string{"Date", "Name", "Active", "Idle"})
			maps.Range(func(name, value interface{}) bool {
				r := value.(*ctxRedis)
				stat := r.pool.Stats()
				c := []string{}
				c = append(c, time.Now().Format("2006-01-02 15:04:05"))
				c = append(c, fmt.Sprintf("%v", r.opts.ServerName))
				c = append(c, fmt.Sprintf("%v", stat.ActiveCount))
				c = append(c, fmt.Sprintf("%v", stat.IdleCount))
				table.Append(c)
				return true
			})
			table.Render()
			log.GenLogf("(Redis Connection Pool Status Information)\n%s", b)
		}
	}()
}

// NewRedis 是redis客户端的初始化函数
// 初始化redis客户端有两种方式， 第一种方式是使用NewRedis函数进行初始化
// 第二种方式是使用rpc-go基础库通过配置文件进行初始化
func NewRedis(o *RedisConfig) (*Redis, error) {
	c := *o
	r, err := newCtxRedis(&c)
	if err != nil {
		return nil, err
	}
	maps.LoadOrStore(o.ServerName, r)
	return &Redis{r}, nil
}

// For 会返回一个具有context的redis客户端， 这个函数
// 是为了支持redis客户端接入trace系统而提供的，参数ctx
// 通常是从基础库中传递下来的。
func (r *Redis) For(ctx context.Context) *ctxRedis {
	return &ctxRedis{
		pool:     r.ctxRedis.pool,
		opts:     r.ctxRedis.opts,
		lastTime: r.ctxRedis.lastTime,
		ctx:      ctx,
	}
}

func newCtxRedis(o *RedisConfig) (*ctxRedis, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	opts := []redis.DialOption{}
	opts = append(opts, redis.DialConnectTimeout(time.Duration(o.ConnectTimeout)*time.Millisecond))
	opts = append(opts, redis.DialReadTimeout(time.Duration(o.ReadTimeout)*time.Millisecond))
	opts = append(opts, redis.DialWriteTimeout(time.Duration(o.WriteTimeout)*time.Millisecond))
	if len(o.Password) != 0 {
		opts = append(opts, redis.DialPassword(o.Password))
	}
	opts = append(opts, redis.DialDatabase(o.Database))
	pool := redisinit(o.Addr, o.Password, o.MaxIdle, o.IdleTimeout, o.MaxActive, opts...)
	go func() {
		for {
			time.Sleep(time.Second * 5)
			stat := pool.Stats()
			utils.ReportEventGauge(
				"redispool.active-count", stat.ActiveCount,
				"name", o.ServerName,
			)
			utils.ReportEventGauge(
				"redispool.idle-count", stat.IdleCount,
				"name", o.ServerName,
			)
		}
	}()
	oo := *o
	return &ctxRedis{
		pool:     pool,
		opts:     &oo,
		lastTime: time.Now().UnixNano(),
		ctx:      context.TODO(),
	}, nil
}

func redisinit(server, password string, maxIdle, idleTimeout, maxActive int, options ...redis.DialOption) *redis.Pool {
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
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

func (r *ctxRedis) RPop(key string) (res string, err error) {
	reply, err := r.do("RPOP", redisBytes, key)
	if err != nil {
		return "", err
	}
	return string(reply.([]byte)[:]), nil
}

func (r *ctxRedis) LPush(name string, fields ...interface{}) error {
	keys := []interface{}{name}
	keys = append(keys, fields...)
	_, err := r.do("LPUSH", nil, keys...)
	return err
}

func (r *ctxRedis) Send(name string, fields ...interface{}) error {
	keys := []interface{}{name}
	keys = append(keys, fields...)
	_, err := r.do("RPUSH", nil, keys...)
	return err
}

func (r *ctxRedis) ReceiveBlock(name string, closech chan struct{}, bufferSize int, block int) chan []byte {
	ch := make(chan []byte, bufferSize)
	go func() {
		defer close(ch)
		for {
			select {
			case <-closech:
				return
			default:
				data, err := r.do("BLPOP", nil, name, block)
				if err == nil {
					if data != nil {
						ms, err := redis.ByteSlices(data, nil)
						if err != nil {
							log.Errorf("convert redis response error %v", err)
						} else {
							ch <- ms[1]
						}
					}
				} else if err != ErrTimeout {
					log.Errorf("BRPOP error %s", err)
				}
			}
		}
	}()
	return ch
}

func (r *ctxRedis) Receive(name string, closech chan struct{}, bufferSize int) chan []byte {
	ch := make(chan []byte, bufferSize)
	go func() {
		defer close(ch)
		for {
			select {
			case <-closech:
				return
			default:
				data, err := r.do("BLPOP", nil, name, 1)
				if err == nil {
					if data != nil {
						ms, err := redis.ByteSlices(data, nil)
						if err != nil {
							log.Errorf("convert redis response error %v", err)
						} else {
							ch <- ms[1]
						}
					}
				} else if err != ErrTimeout {
					log.Errorf("BRPOP error %s", err)
				}
			}
		}
	}()
	return ch
}

// Do 函数为通用的函数， 可以执行任何redis服务器支持的命令
func (r *ctxRedis) Do(cmd string, args ...interface{}) (reply interface{}, err error) {
	return r.do(cmd, nil, args...)
}

// DoCtx 函数与Do函数相比， 增加了一个context参数， 提供了超时的功能
func (r *ctxRedis) DoCtx(ctx context.Context, cmd string, args ...interface{}) (interface{}, error) {
	var (
		ch    = make(chan struct{})
		reply interface{}
		err   error
	)
	go func() {
		defer close(ch)
		reply, err = r.do(cmd, nil, args...)
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ch:
		return reply, err
	}
}

// Set 返回两个参数，err不为空为服务器内服错误， 当命令执行成功时，ret为true
// 使用这个函数时需要同时判断这两个返回值
func (r *ctxRedis) Set(key, value interface{}) (ret bool, err error) {
	var reply interface{}
	reply, err = r.do("SET", redisString, key, value)
	if err != nil {
		return
	}
	rsp := reply.(string)

	if rsp == "OK" {
		ret = true
	}

	return
}

func (r *ctxRedis) SetExSecond(key, value interface{}, dur int) (ret string, err error) {
	var reply interface{}
	reply, err = r.do("SET", redisString, key, value, "EX", dur)
	if err != nil {
		return
	}
	ret = reply.(string)
	return
}

func (r *ctxRedis) Get(key string) (ret []byte, err error) {
	var reply interface{}
	reply, err = r.do("GET", redisBytes, key)
	if err != nil {
		if err == redis.ErrNil {
			err = nil
			var tmp []byte
			ret = tmp
		}
		return
	}
	ret = reply.([]byte)
	return
}

func (r *ctxRedis) GetInt(key string) (ret int, err error) {
	var reply interface{}
	reply, err = r.do("GET", redisInt, key)
	if err != nil {
		return
	}
	ret = reply.(int)
	return
}

func (r *ctxRedis) MGet(keys ...interface{}) (ret [][]byte, err error) {
	var reply interface{}
	reply, err = r.do("MGET", redisByteSlices, keys...)
	if err != nil {
		return
	}
	ret = reply.([][]byte)
	return
}

func (r *ctxRedis) MSet(keys ...interface{}) (ret string, err error) {
	var reply interface{}
	reply, err = r.do("MSET", redisString, keys...)
	if err != nil {
		return
	}
	ret = reply.(string)
	return
}

func (r *ctxRedis) Del(args ...interface{}) (count int, err error) {
	var reply interface{}
	reply, err = r.do("Del", redisInt, args...)
	if err != nil {
		return
	}
	count = reply.(int)
	return
}

func (r *ctxRedis) Exists(key string) (res bool, err error) {
	var reply interface{}
	reply, err = r.do("Exists", redisBool, key)
	if err != nil {
		return
	}
	res = reply.(bool)
	return
}

func (r *ctxRedis) Expire(key string, expire time.Duration) error {
	_, err := r.do("EXPIRE", nil, key, int64(expire.Seconds()))
	if err != nil {
		return err
	}
	return nil
}

/*
*	hash
 */
func (r *ctxRedis) HDel(key interface{}, fields ...interface{}) (res int, err error) {
	var reply interface{}
	keys := []interface{}{key}
	keys = append(keys, fields...)

	reply, err = r.do("HDEL", redisInt, keys...)
	if err != nil {
		return
	}
	res = reply.(int)
	return
}

func (r *ctxRedis) HSet(key, fieldk string, fieldv interface{}) (res int, err error) {
	var reply interface{}
	reply, err = r.do("HSET", redisInt, key, fieldk, fieldv)
	if err != nil {
		return
	}
	res = reply.(int)
	return
}

func (r *ctxRedis) HGet(key, field string) (res string, err error) {
	var reply interface{}
	reply, err = r.do("HGET", redisString, key, field)
	if err != nil {
		return
	}
	res = reply.(string)
	return
}

func (r *ctxRedis) HGetInt(key, field string) (res int, err error) {
	var reply interface{}
	reply, err = r.do("HGET", redisInt, key, field)
	if err != nil {
		return
	}
	res = reply.(int)
	return
}

func (r *ctxRedis) HMGet(key string, fields ...interface{}) (res []string, err error) {
	var reply interface{}
	keys := []interface{}{key}
	keys = append(keys, fields...)
	reply, err = r.do("HMGET", redisStrings, keys...)
	if err != nil {
		return
	}
	res = reply.([]string)
	return
}

func (r *ctxRedis) HMSet(key string, fields ...interface{}) (res string, err error) {
	var reply interface{}
	keys := []interface{}{key}
	keys = append(keys, fields...)
	reply, err = r.do("HMSET", redisString, keys...)
	if err != nil {
		return
	}
	res = reply.(string)
	return
}

func (r *ctxRedis) HGetAll(key string) (res map[string]string, err error) {
	var reply interface{}
	reply, err = r.do("HGETALL", redisStringMap, key)
	if err != nil {
		return
	}
	res = reply.(map[string]string)
	return
}

func (r *ctxRedis) HKeys(key string) (res []string, err error) {
	var reply interface{}
	reply, err = r.do("HKEYS", redisStrings, key)
	if err != nil {
		return
	}
	res = reply.([]string)
	return
}

func (r *ctxRedis) HIncrby(key, field string, incr int) (res int64, err error) {
	var reply interface{}
	reply, err = r.do("HINCRBY", redisInt64, key, field, incr)
	if err != nil {
		return
	}
	res = reply.(int64)
	return
}

/*
*	set
 */
func (r *ctxRedis) SAdd(key string, members ...interface{}) (res int, err error) {
	var reply interface{}
	keys := []interface{}{key}
	keys = append(keys, members...)
	reply, err = r.do("SADD", redisInt, keys...)
	if err != nil {
		return
	}
	res = reply.(int)
	return
}

func (r *ctxRedis) SRem(key string, members ...interface{}) (res int, err error) {
	var reply interface{}
	keys := []interface{}{key}
	keys = append(keys, members...)
	reply, err = r.do("SREM", redisInt, keys...)
	if err != nil {
		return
	}
	res = reply.(int)
	return
}

func (r *ctxRedis) SIsMember(key string, member string) (res bool, err error) {
	var reply interface{}
	reply, err = r.do("SISMEMBER", redisBool, key, member)
	if err != nil {
		return
	}
	res = reply.(bool)

	return
}

func (r *ctxRedis) SMembers(key string) (res []string, err error) {
	var reply interface{}
	reply, err = r.do("SMEMBERS", redisStrings, key)
	if err != nil {
		return
	}
	res = reply.([]string)
	return
}

func (r *ctxRedis) ZAdd(key string, args ...interface{}) (res int, err error) {
	var reply interface{}
	keys := []interface{}{key}
	keys = append(keys, args...)
	reply, err = r.do("ZADD", redisInt, keys...)
	if err != nil {
		return
	}
	res = reply.(int)
	return
}

func (r *ctxRedis) ZRange(key string, args ...interface{}) (res []string, err error) {
	var reply interface{}
	keys := []interface{}{key}
	keys = append(keys, args...)
	reply, err = r.do("ZRANGE", redisStrings, keys...)
	if err != nil {
		return
	}
	res = reply.([]string)
	return
}

func (r *ctxRedis) ZRangeInt(key string, start, stop int) (res []int, err error) {
	var reply interface{}
	reply, err = r.do("ZRANGE", redisInts, key, start, stop)
	if err != nil {
		return
	}
	res = reply.([]int)
	return
}

func (r *ctxRedis) ZRangeWithScore(key string, start, stop int) (res []string, err error) {
	var reply interface{}
	reply, err = r.do("ZRANGE", redisStrings, key, start, stop, "WITHSCORES")
	if err != nil {
		return
	}
	res = reply.([]string)
	return
}

func (r *ctxRedis) ZRevRangeWithScore(key string, start, stop int) (res []string, err error) {
	var reply interface{}
	reply, err = r.do("ZREVRANGE", redisStrings, key, start, stop, "WITHSCORES")
	if err != nil {
		return
	}
	res = reply.([]string)
	return
}

func (r *ctxRedis) ZCount(key string, min, max int) (res int, err error) {
	var reply interface{}
	reply, err = r.do("ZCOUNT", redisInt, key, min, max)
	if err != nil {
		return
	}
	res = reply.(int)
	return
}

func (r *ctxRedis) ZCard(key string) (res int, err error) {
	var reply interface{}
	reply, err = r.do("ZCARD", redisInt, key)
	if err != nil {
		return
	}
	res = reply.(int)
	return
}

func (r *ctxRedis) LLen(key string) (res int64, err error) {
	var reply interface{}
	reply, err = r.do("LLEN", redisInt64, key)
	if err != nil {
		return
	}
	res = reply.(int64)
	return
}

func (r *ctxRedis) Incrby(key string, incr int) (res int64, err error) {
	var reply interface{}
	reply, err = r.do("INCRBY", redisInt64, key, incr)
	if err != nil {
		return
	}
	res = reply.(int64)
	return
}

func (r *ctxRedis) ZIncrby(key string, incr int, member string) (res int, err error) {
	var reply interface{}
	reply, err = r.do("ZINCRBY", redisInt, key, incr, member)
	if err != nil {
		return
	}
	res = reply.(int)
	return
}

/*
* If the member not in the zset or key not exits, ZRank will return ErrNil
 */
func (r *ctxRedis) ZRank(key string, member string) (res int, err error) {
	var reply interface{}
	reply, err = r.do("ZRANK", redisInt, key, member)
	if err != nil {
		return
	}
	res = reply.(int)
	return
}

/*
* If the members not in the zset or key not exits, ZRem will return ErrNil
 */
func (r *ctxRedis) ZRem(key string, members ...interface{}) (res int, err error) {
	var reply interface{}
	keys := []interface{}{key}
	keys = append(keys, members...)

	reply, err = r.do("ZREM", redisInt, keys...)
	if err != nil {
		return
	}
	res = reply.(int)
	return
}

func (r *ctxRedis) ZRemrangebyrank(key string, members ...interface{}) (res int, err error) {
	var reply interface{}
	keys := []interface{}{key}
	keys = append(keys, members...)

	reply, err = r.do("ZREMRANGEBYRANK", redisInt, keys...)
	if err != nil {
		return
	}
	res = reply.(int)
	return
}

func (r *ctxRedis) Subscribe(ctx context.Context, key string, maxSize int) (chan []byte, error) {
	ch := make(chan []byte, maxSize)

	if r.opts.ReadTimeout < 100 && r.opts.ReadTimeout > 0 {
		return nil, errors.New("Read timeout should be longer")
	}

	healthCheckPeriod := r.opts.ReadTimeout * (70 / 100) // 70%

	var offHealthCheck = (healthCheckPeriod == 0)
	done := make(chan error, 1)

	// While not a permanent error on the connection.
	go func() {
	start:
		client := r.pool.Get()
		psc := redis.PubSubConn{client}
		// Set up subscriptions
		err := psc.Subscribe(key)
		if err != nil {
			client.Close()
			close(ch)
			return
		}

		go func(c redis.PubSubConn) {
			if offHealthCheck {
				return
			}
			ticker := time.NewTicker(time.Duration(healthCheckPeriod * 10e5))
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err = c.Ping(""); err != nil {
						break
					}
				case <-ctx.Done():
					return
				case <-done:
					return
				}
			}
		}(psc)

		for client.Err() == nil {
			select {
			case <-ctx.Done():
				client.Close()
				return
			default:
				switch v := psc.ReceiveWithTimeout(time.Second * 0).(type) {
				case redis.Message:
					ch <- v.Data
				case redis.Subscription:
					log.Infof("Receive chan (%s) %s %d", v.Channel, v.Kind, v.Count)
				case error:
					log.Errorf("Receive error (%v), client will reconnect..", v)
					client.Close()
					if !offHealthCheck {
						done <- v
					}
					time.Sleep(time.Second / 10)
					goto start
				}
			}
		}
	}()

	return ch, nil
}

/*
* If the member not in the zset or key not exits, ZScore will return ErrNil
 */
func (r *ctxRedis) ZScore(key, member string) (res float64, err error) {
	var reply interface{}
	reply, err = r.do("ZSCORE", redisFloat64, key, member)
	if err != nil {
		return
	}
	res = reply.(float64)
	return
}

func (r *ctxRedis) Zrevrange(key string, args ...interface{}) (res []string, err error) {
	var reply interface{}
	argss := []interface{}{key}
	argss = append(argss, args...)
	reply, err = r.do("ZREVRANGE", redisStrings, argss...)
	if err != nil {
		return
	}
	res = reply.([]string)
	return
}

func (r *ctxRedis) Zrevrangebyscore(key string, args ...interface{}) (res []string, err error) {
	var reply interface{}
	argss := []interface{}{key}
	argss = append(argss, args...)
	reply, err = r.do("ZREVRANGEBYSCORE", redisStrings, argss...)
	if err != nil {
		return
	}
	res = reply.([]string)
	return
}

func (r *ctxRedis) ZrevrangebyscoreInt(key string, args ...interface{}) (res []int, err error) {
	var reply interface{}
	argss := []interface{}{key}
	argss = append(argss, args...)
	reply, err = r.do("ZREVRANGEBYSCORE", redisInts, argss...)
	if err != nil {
		return
	}
	res = reply.([]int)
	return
}

func (r *ctxRedis) TryLock(key string, acquireTimeout, expireTimeout time.Duration) (uuid string, err error) {
	deadline := time.Now().Add(acquireTimeout)
	for {
		if time.Now().After(deadline) {
			return "", errors.New("lock timeout")
		}
		uuid, err = r.Lock(key, expireTimeout)
		if err != nil {
			time.Sleep(time.Millisecond)
		} else {
			return uuid, err
		}
	}
}
func (r *ctxRedis) Lock(key string, expire time.Duration) (uuid string, err error) {
	buf := make([]byte, 16)
	_, _ = io.ReadFull(crand.Reader, buf)
	uuid = hex.EncodeToString(buf)
	ret, err := redis.String(r.Do("SET", key, uuid, "NX", "PX", expire.Nanoseconds()/1e6))
	if ret != "OK" {
		return "", errors.New("lock failed")
	}
	return
}
func (r *ctxRedis) Unlock(key string, uuid string) (err error) {
	script := `if redis.call('get',KEYS[1]) == ARGV[1] then return redis.call('del',KEYS[1]) else return 0 end`
	ret, err := redis.Int(r.Do("EVAL", script, 1, key, uuid))
	if ret != 1 {
		return errors.New("unlock failed")
	}
	return
}

// Pipelining
func (r *ctxRedis) randomDuration(n int64) time.Duration {
	s := rand.NewSource(r.lastTime)
	return time.Duration(rand.New(s).Int63n(n) + 1)
}

var statIgnoredCMD = map[string]bool{
	"BLPOP": true,
	"BRPOP": true,
}

func (r *ctxRedis) do(cmd string, f func(interface{}, error) (interface{}, error), args ...interface{}) (reply interface{}, err error) {
	var (
		stCode   = redisSuccess
		st       = utils.NewServiceStatEntry("reclient", r.opts.ServerName)
		count    = 0
		now      = time.Now()
		address  = r.opts.Addr
		fristArg interface{}
	)

	if len(args) >= 1 {
		fristArg = args[0]
	}

	timeout := -1
	needIgnored := false
	cmdCase := strings.ToUpper(cmd)
	if _, ok := statIgnoredCMD[cmdCase]; ok {
		needIgnored = true
		if tm, ok := args[len(args)-1].(int); ok {
			timeout = tm
		}
	}
	myCtx := r.ctx
	if myCtx == nil || myCtx == context.TODO() || myCtx == context.Background() {
		if ctx, ok := tls.GetContext(); ok {
			myCtx = ctx
		}
	}
	span, _ := opentracing.StartSpanFromContext(
		myCtx,
		fmt.Sprintf("REDIS Client Cmd %s", cmd),
	)

	traceid := strings.SplitN(fmt.Sprintf("%s", span.Context()), ":", 2)[0]

	// annotation
	ext.SpanKindRPCClient.Set(span)
	ext.PeerAddress.Set(span, address)
	ext.PeerService.Set(span, r.opts.ServerName)
	ext.DBType.Set(span, "codis/reids")
	ext.Component.Set(span, "golang/redis-client")
	span.SetTag("slowtime", r.opts.SlowTime)

	defer func() {
		if !needIgnored {
			st.End(cmd, stCode)
		}
		atomic.StoreInt64(&r.lastTime, time.Now().UnixNano())
		span.SetTag("errcode", stCode)

		if err != nil {
			span.LogFields(opentracinglog.Error(err))
			ext.Error.Set(span, true)
		}

		span.Finish()
	}()

retry1:
	client := r.pool.Get()
	defer client.Close()

	reply, err = client.Do(cmd, args...)
	if f != nil {
		reply, err = f(reply, err)
	}
	if err == redis.ErrNil {
		err = nil
	}

	// if err is not redis.Error, it will retry
	if _, ok := err.(redis.Error); err != nil && !ok {
		var rterr error
		switch err {
		case redis.ErrPoolExhausted:
			stCode = redisConnExhausted
			rterr = ErrConnExhausted
		default:
			if strings.Contains(err.Error(), "timeout") {
				stCode = redisTimeout
				rterr = ErrTimeout
			} else {
				stCode = redisError
				rterr = err
			}
		}

		if r.opts.Retry > 0 && count < r.opts.Retry {
			count++
			log.GenLogf(
				"%d|redisclient|%s|retry-%d|%s|%s|%v|%d|%v|%s",
				serverLocalPid, r.opts.ServerName, count,
				traceid, cmd, fristArg, stCode, err, address,
			)
			span.LogFields(
				opentracinglog.Int("retry", count),
				opentracinglog.String("cmd", cmd),
				opentracinglog.String("cause", err.Error()),
				opentracinglog.Int("code", stCode),
			)
			time.Sleep(time.Millisecond * r.randomDuration(10))
			goto retry1
		}
		log.GenLogf(
			"%d|redisclient|%s|exceed-retry|%d|%s|%s|%v|%d|%v|%s",
			serverLocalPid, r.opts.ServerName, count,
			traceid, cmd, fristArg, stCode, err, address,
		)
		return nil, rterr
	}

	endTime := time.Now()
	costTime := endTime.Sub(now).Nanoseconds() / int64(time.Millisecond)

	span.LogFields(opentracinglog.String("event", "Do done"))

	if err != nil {
		log.GenLogf(
			"%d|redisclient|%s|%s|%v|%d|%d|%v|%s",
			serverLocalPid, traceid, cmd, fristArg,
			stCode, costTime, err, address,
		)
	}

	if (r.opts.SlowTime > 0 && costTime > int64(r.opts.SlowTime)) || (stCode == redisTimeout) {
		log.SlowLogf(
			"%d|%s|redisclient|%s|%s|%s|%v|%d|%d|%s|%v",
			serverLocalPid, endTime.Format(logFormat), r.opts.ServerName,
			traceid, cmd, fristArg, stCode, costTime, address, err,
		)
		span.LogFields(
			opentracinglog.Bool("Slow", true),
		)
	}

	timeoutMs := int64(timeout) * int64(time.Second) / int64(time.Millisecond)
	if needIgnored && stCode == redisSuccess {
		if timeout > 0 && costTime >= timeoutMs { //忽略上报,BLPOP/BRPOP操作超时
			needIgnored = true
		} else if timeout == 0 { //忽略上报,配置成一直阻塞,并成功返回了
			needIgnored = true
		} else if costTime < timeoutMs && timeoutMs-costTime >= timeoutMs*3/10 { //忽略上报,耗时在合理范围内
			needIgnored = true
		} else {
			needIgnored = false
		}
	}
	return
}
