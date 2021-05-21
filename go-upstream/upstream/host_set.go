package upstream

import (
	"sync"
	"sync/atomic"
)

type memberUpdateCallback func(added, removed []*Host)

type HostSet struct {
	hosts       atomic.Value
	healthHosts atomic.Value
	callbacks   []memberUpdateCallback
	mutex       sync.Mutex
}

func NewHostSet(hosts, healthHosts []*Host) *HostSet {
	hs := &HostSet{
		callbacks: make([]memberUpdateCallback, 0),
	}
	hs.hosts.Store(hosts)
	hs.healthHosts.Store(healthHosts)
	return hs
}

func (h *HostSet) Hosts() []*Host {
	return h.hosts.Load().([]*Host)
}

func (h *HostSet) HelathHosts() []*Host {
	return h.healthHosts.Load().([]*Host)
}

func (h *HostSet) UpdateHosts(hosts, healthHosts, added, removed []*Host) {
	h.hosts.Store(hosts)
	h.healthHosts.Store(healthHosts)
	h.runUpdateCallback(added, removed)
}

func (h *HostSet) AddUpdateCallback(m memberUpdateCallback) {
	h.mutex.Lock()
	h.callbacks = append(h.callbacks, m)
	h.mutex.Unlock()
}

func (h *HostSet) runUpdateCallback(added, removed []*Host) {
	callbacks := make([]memberUpdateCallback, 0, len(h.callbacks))
	h.mutex.Lock()
	callbacks = append(callbacks, h.callbacks...)
	h.mutex.Unlock()
	for _, c := range callbacks {
		c(added, removed)
	}
}
