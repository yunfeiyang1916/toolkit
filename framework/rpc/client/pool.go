package client

import (
	"sync"
	"time"

	"github.com/jonboulle/clockwork"
)

type pool struct {
	size    int
	ttl     int64
	cleanup int64
	d       dialer

	// protect sockets
	sync.Mutex
	sockets map[string]map[socket]int64

	clock clockwork.Clock
}

func newPool(size int, d dialer, ttl time.Duration, clock clockwork.Clock) *pool {
	if clock == nil {
		clock = clockwork.NewRealClock()
	}
	return &pool{
		ttl:     int64(ttl.Seconds()),
		size:    size,
		sockets: make(map[string]map[socket]int64),
		cleanup: clock.Now().Unix(),
		d:       d,
		clock:   clock,
	}
}

func (p *pool) Close() error {
	// TODO
	return nil
}

func (p *pool) release(host string, sock socket, err error) {
	if err == nil {
		return
	}

	p.Lock()
	defer p.Unlock()

	if sockets := p.sockets[host]; sockets != nil {
		if _, ok := sockets[sock]; ok {
			sock.Close()
			delete(sockets, sock)
		}
	}
}

func (p *pool) getSocket(host string) (socket, error) {
	p.Lock()
	defer p.Unlock()

	now := p.clock.Now().Unix()

	// check wheather if it's time to clien up all old socket.
	if now-p.cleanup > 60 {
		p.cleanup = now
		for _, sockets := range p.sockets {
			for sock, accessed := range sockets {
				if d := now - accessed; d > p.ttl {
					delete(sockets, sock)
					go sock.Close()
				}
			}
		}
	}

	// if current size is smaller then pool size, we create new sock.
	if len(p.sockets[host]) < p.size {
		// release lock
		p.Unlock()
		sock, err := p.d.Dial(host)
		p.Lock()

		if err != nil {
			return nil, err
		}

		if p.sockets[host] == nil {
			p.sockets[host] = make(map[socket]int64)
		}
		p.sockets[host][sock] = p.clock.Now().Unix()
		return sock, nil
	}

	// release old sock.
	for sock, accessed := range p.sockets[host] {
		if d := now - accessed; d > p.ttl {
			//	sock.Close()
			continue
		}
		p.sockets[host][sock] = p.clock.Now().Unix()
		return sock, nil
	}

	p.Unlock()
	// create new sock.
	sock, err := p.d.Dial(host)
	p.Lock()
	if err != nil {
		return nil, err
	}

	p.sockets[host][sock] = p.clock.Now().Unix()
	return sock, nil
}
