package session

import (
	"sync"
	"sync/atomic"

	"github.com/yunfeiyang1916/toolkit/ikio"
)

type Session struct {
	conn   *ikio.ServerChannel
	ConnID int64

	info         interface{}
	mu           sync.RWMutex
	lastHearbeat int64
}

func NewSession(c *ikio.ServerChannel) *Session {
	return &Session{
		conn:   c,
		ConnID: c.ConnID(),
	}
}

func (s *Session) Conn() *ikio.ServerChannel {
	return s.conn
}

func (s *Session) SetInfo(info interface{}) {
	s.mu.Lock()
	s.info = info
	s.mu.Unlock()
}

func (s *Session) Info() interface{} {
	s.mu.RLock()
	info := s.info
	s.mu.RUnlock()
	return info
}

func (s *Session) UpdateInfo(cb func(info interface{})) {
	s.mu.Lock()
	cb(s.info)
	s.mu.Unlock()
}

func (s *Session) UpdateLastHeartbeat(t int64) {
	atomic.StoreInt64(&s.lastHearbeat, t)
}

func (s *Session) LastHeartbeat() int64 {
	return atomic.LoadInt64(&s.lastHearbeat)
}
