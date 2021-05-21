package client

import (
	"sync"
)

type Factory interface {
	Client(endpoint string) Client
}

type sfactory struct {
	opts  []Option
	cache sync.Map
	mu    sync.Mutex
}

func SFactory(opts ...Option) Factory {
	return &sfactory{opts: opts}
}

func (f *sfactory) Client(endpoint string) Client {
	if c, ok := f.cache.Load(endpoint); ok {
		return c.(Client)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.cache.Load(endpoint); ok {
		return c.(Client)
	}
	client := SClient(endpoint, f.opts...)
	f.cache.Store(endpoint, client)
	return client
}

type hfactory struct {
	opts  []Option
	cache sync.Map
	mu    sync.Mutex
}

func HFactory(opts ...Option) Factory {
	return &hfactory{opts: opts}
}

func (f *hfactory) Client(endpoint string) Client {
	if c, ok := f.cache.Load(endpoint); ok {
		return c.(Client)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.cache.Load(endpoint); ok {
		return c.(Client)
	}
	client := HClient(endpoint, f.opts...)
	f.cache.Store(endpoint, client)
	return client
}
