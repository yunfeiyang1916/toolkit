// Package memory is a memory source
package memory

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yunfeiyang1916/toolkit/framework/config/source"
)

type memory struct {
	sync.RWMutex
	ChangeSet *source.ChangeSet
	Watchers  map[string]*watcher
}

func (s *memory) Read() (*source.ChangeSet, error) {
	cs := &source.ChangeSet{
		Timestamp: s.ChangeSet.Timestamp,
		Data:      s.ChangeSet.Data,
		Checksum:  s.ChangeSet.Checksum,
		Source:    s.ChangeSet.Source,
		Format:    s.ChangeSet.Format,
	}
	return cs, nil
}
func (s *memory) Watch() (source.Watcher, error) {
	w := &watcher{
		Id:      uuid.New().String(),
		Updates: make(chan *source.ChangeSet, 100),
		Source:  s,
	}

	s.Lock()
	s.Watchers[w.Id] = w
	s.Unlock()
	return w, nil
}
func (s *memory) String() string {
	return "memory"
}

// Update allows manual updates of the config data.
func (s *memory) Update(c *source.ChangeSet) {
	if c == nil {
		return
	}
	s.Lock()
	s.ChangeSet = &source.ChangeSet{
		Data:      c.Data,
		Format:    c.Format,
		Source:    "memory",
		Timestamp: time.Now(),
	}
	s.ChangeSet.Checksum = s.ChangeSet.Sum()

	// update watchers
	for _, w := range s.Watchers {
		select {
		case w.Updates <- s.ChangeSet:
		default:
		}
	}
	s.Unlock()
}

func NewSource(opts ...source.Option) source.Source {
	var options source.Options
	for _, o := range opts {
		o(&options)
	}

	s := &memory{
		Watchers: make(map[string]*watcher),
	}

	if options.Context != nil {
		if c, ok := options.Context.Value(rawChangeSetKey{}).(*source.ChangeSet); ok {
			s.Update(c)
		}
		if c, ok := options.Context.Value(jsonChangeSetKey{}).(*source.ChangeSet); ok {
			s.Update(c)
		}
		if c, ok := options.Context.Value(tomlChangeSetKey{}).(*source.ChangeSet); ok {
			s.Update(c)
		}
	}
	return s
}
