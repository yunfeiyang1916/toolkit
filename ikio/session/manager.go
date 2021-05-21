package session

import (
	"sync"
	"sync/atomic"

	"github.com/yunfeiyang1916/toolkit/ikio"
	"golang.org/x/net/context"
)

type Manager struct {
	sessions *sync.Map
	total    int64
}

func NewManager() *Manager {
	return &Manager{
		sessions: new(sync.Map),
	}
}

func (sm *Manager) Delete(id int64) {
	atomic.AddInt64(&sm.total, -1)
	sm.sessions.Delete(id)
}

func (sm *Manager) Set(id int64, s *Session) {
	atomic.AddInt64(&sm.total, 1)
	sm.sessions.Store(id, s)
}

func (sm *Manager) Get(id int64) *Session {
	s, ok := sm.sessions.Load(id)
	if !ok {
		return nil
	}
	return s.(*Session)
}

func (sm *Manager) Size() int64 {
	return atomic.LoadInt64(&sm.total)
}

func (sm *Manager) Run(ctx context.Context, wc ikio.WriteCloser, cb func(ctx context.Context, s *Session)) bool {
	conn := wc.(*ikio.ServerChannel)
	s, ok := sm.sessions.Load(conn.ConnID())
	if !ok {
		return false
	}
	cb(ctx, s.(*Session))
	return true
}
