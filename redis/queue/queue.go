package queue

import (
	"github.com/garyburd/redigo/redis"
	log "github.com/yunfeiyang1916/toolkit/logging"
	helper "github.com/yunfeiyang1916/toolkit/redis"
)

/*
*   Waraing: APIs in this file is deprecated.
 */

type RedisQueueConfig struct {
	Addr           string `json:"addr"`
	Password       string `json:"password"`
	QueueName      string `json:"queue_name"`
	QueueBlock     int    `json:"queue_block"`
	ConnectTimeout int    `json:"connect_timeout"`
}

type RedisQueue struct {
	serverAddr     string
	serverPassword string
	blockTimeout   int
	queueName      string
	pool           *redis.Pool
	messageChan    chan []byte
}

func NewRedisQueue(conf RedisQueueConfig) *RedisQueue {
	queue := RedisQueue{serverAddr: conf.Addr, serverPassword: conf.Password, blockTimeout: conf.QueueBlock, queueName: conf.QueueName}
	poolConf := helper.RedisPoolConfig{Addr: conf.Addr, Password: conf.Password, MaxIdle: 100, IdleTimeout: 120000, MaxActive: 200, ConnectTimeout: conf.ConnectTimeout, ReadTimeout: 0, WriteTimeout: 0}
	queue.pool = helper.RedisPoolInit(&poolConf)
	return &queue
}

func (q *RedisQueue) Send(bytes []byte) error {
	client := q.pool.Get()
	defer client.Close()
	_, err := redis.Int(client.Do("LPUSH", q.queueName, bytes))
	return err
}

func (q *RedisQueue) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	client := q.pool.Get()
	defer client.Close()
	reply, err = client.Do(commandName, args...)
	return
}

func (q *RedisQueue) Messages(closeChan chan bool, maxQueueSize int) chan []byte {
	ch := make(chan []byte, maxQueueSize)
	go func() {
		for {
			select {
			case <-closeChan:
				close(ch)
				return
			default:
				client := q.pool.Get()
				data, err := client.Do("BRPOP", q.queueName, q.blockTimeout)
				if err == nil {
					if data != nil {
						ms, err := redis.ByteSlices(data, nil)
						if err != nil {
							log.Errorf("convert redis response error %v", err)
						} else {
							ch <- ms[1]

						}
					}
				} else {
					log.Errorf("BRPOP error %s", err)
				}
				client.Close()
			}
		}
	}()
	return ch
}
