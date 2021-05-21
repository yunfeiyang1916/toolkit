package config

import (
	"path"
	"strings"
	"sync"
	"time"

	"github.com/yunfeiyang1916/toolkit/framework/config/loader"
	"github.com/yunfeiyang1916/toolkit/framework/config/loader/memory"
	"github.com/yunfeiyang1916/toolkit/framework/config/reader"
	"github.com/yunfeiyang1916/toolkit/framework/config/reader/toml"
	"github.com/yunfeiyang1916/toolkit/framework/config/source"
	"github.com/yunfeiyang1916/toolkit/framework/config/source/consul"
	"github.com/yunfeiyang1916/toolkit/framework/config/source/file"
)

type defaultConfig struct {
	sync.RWMutex
	opts   Options
	snap   *loader.Snapshot
	vals   reader.Values
	loader loader.Loader // use memory loader
	reader reader.Reader // default use toml reader to read data from memory loader
}

func newDefaultConfig(opts ...Option) *defaultConfig {
	ops := Options{}
	for _, o := range opts {
		o(&ops)
	}
	c := &defaultConfig{
		loader: memory.NewLoader(),
		reader: toml.NewReader(),
	}
	c.loader.Load(ops.Source...)
	snap, err := c.loader.Snapshot()
	if err != nil {
		panic(err)
	}
	vals, err := c.reader.Values(snap.ChangeSet)
	if err != nil {
		panic(err)
	}
	c.opts = ops
	c.snap = snap
	c.vals = vals
	go c.run()
	return c
}

func (c *defaultConfig) LoadFile(files ...string) error {
	s := make([]source.Source, len(files))
	for i, f := range files {
		if len(f) == 0 {
			return nil
		}
		enc := TomlEncoder()
		fn := path.Base(f)
		suffix := path.Ext(fn)
		if e := reader.Encoding[strings.TrimLeft(suffix, ".")]; e != nil {
			enc = e
		}
		s[i] = file.NewSource(file.WithPath(f), source.WithEncoder(enc))
	}
	return c.load(s...)
}

func (c *defaultConfig) LoadPath(path string, usePrefix bool, format string) error {
	if len(path) == 0 {
		return nil
	}
	s := consul.NewSource(
		consul.WithAddress(ConsulAddr),
		consul.WithAbsPath(path),
		consul.UsePrefix(usePrefix),
		source.WithEncoder(reader.Encoder(format)))
	return c.load(s)
}

func (c *defaultConfig) Sync() error {
	if err := c.loader.Sync(); err != nil {
		return err
	}
	return c.update()
}

func (c *defaultConfig) Listen(v interface{}) loader.Refresher {
	c.Lock()
	defer c.Unlock()
	return c.loader.Listen(v)
}

// implements reader.Values
func (c *defaultConfig) Bytes() []byte {
	c.RLock()
	defer c.RUnlock()

	if c.vals == nil {
		return []byte{}
	}
	return c.vals.Bytes()
}

func (c *defaultConfig) String() string {
	c.RLock()
	defer c.RUnlock()
	if c.vals == nil {
		return ""
	}
	return c.vals.String()
}

func (c *defaultConfig) Get(keys ...string) reader.Value {
	c.RLock()
	defer c.RUnlock()
	if c.vals != nil {
		return c.vals.Get(keys...)
	}
	return nil
}

func (c *defaultConfig) Map() map[string]interface{} {
	c.RLock()
	defer c.RUnlock()
	return c.vals.Map()
}

func (c *defaultConfig) Range(f func(k string, v interface{})) {
	m := c.Map()
	for k, v := range m {
		kk, vv := k, v
		f(kk, vv)
	}
}

func (c *defaultConfig) Scan(v interface{}) error {
	c.Lock()
	defer c.Unlock()
	return c.vals.Scan(v)
}

func (c *defaultConfig) load(sources ...source.Source) error {
	if err := c.loader.Load(sources...); err != nil {
		return err
	}
	return c.update()
}

func (c *defaultConfig) update() error {
	snap, err := c.loader.Snapshot()
	if err != nil {
		return err
	}
	c.Lock()
	defer c.Unlock()

	c.snap = snap
	vals, err := c.reader.Values(snap.ChangeSet)
	if err != nil {
		return err
	}
	c.vals = vals
	return nil
}

func (c *defaultConfig) run() {
	watch := func(w loader.Watcher) error {
		for {
			snap, err := w.Next()
			if err != nil {
				return err
			}
			c.Lock()
			c.snap = snap
			c.vals, _ = c.reader.Values(snap.ChangeSet)
			c.Unlock()
		}
	}

	for {
		// memory loader's watcher, watch integral config changing, not sub key
		w, err := c.loader.Watch()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		done := make(chan bool)
		go func() {
			<-done
			w.Stop()
		}()

		// block watch
		if err := watch(w); err != nil {
			time.Sleep(time.Second)
		}
		close(done)
	}
}
