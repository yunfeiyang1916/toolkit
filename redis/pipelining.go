package redis

import (
	"context"
	"errors"
	"sync"

	"github.com/garyburd/redigo/redis"
)

// Pipelining 提供了一些流水线的一些方法, 由NewPipelining函数创建
type Pipelining struct {
	conn    redis.Conn
	mu      sync.Mutex
	isClose bool
}

// NewPipelining函数创建一个Pipelining， 参数ctx用于trace系统
func (r *Redis) NewPipelining(ctx context.Context) (*Pipelining, error) {
	p := &Pipelining{}
	client := r.pool.Get()
	err := client.Err()
	if err != nil {
		return nil, err
	}
	p.conn = client
	p.mu = sync.Mutex{}
	return p, nil
}

func (p *Pipelining) Send(cmd string, args ...interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.isClose {
		return errors.New("Pipelining closed")
	}
	return p.conn.Send(cmd, args...)
}

func (p *Pipelining) Flush() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.isClose {
		return errors.New("Pipelining closed")
	}
	return p.conn.Flush()
}

func (p *Pipelining) Receive() (reply interface{}, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.isClose {
		return nil, errors.New("Pipelining closed")
	}
	return p.conn.Receive()
}

func (p *Pipelining) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.isClose = true
	return p.conn.Close()
}
