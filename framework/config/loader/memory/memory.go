package memory

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/yunfeiyang1916/toolkit/logging"

	"github.com/yunfeiyang1916/toolkit/framework/config/loader"
	"github.com/yunfeiyang1916/toolkit/framework/config/reader"
	"github.com/yunfeiyang1916/toolkit/framework/config/reader/toml"
	"github.com/yunfeiyang1916/toolkit/framework/config/source"
)

type memory struct {
	sync.RWMutex
	exit     chan bool
	lisChan  chan bool
	opts     loader.Options
	snap     *loader.Snapshot
	vals     reader.Values
	sets     []*source.ChangeSet
	sources  []source.Source
	idx      int
	watchers map[int]*watcher
}

func NewLoader(opts ...loader.Option) loader.Loader {
	options := loader.Options{
		Reader: toml.NewReader(),
	}
	for _, o := range opts {
		o(&options)
	}
	m := &memory{
		exit:     make(chan bool),
		lisChan:  make(chan bool),
		opts:     options,
		watchers: make(map[int]*watcher),
		sources:  options.Source,
	}
	for i, s := range options.Source {
		go m.watch(i, s) // 启动每个资源的watcher
	}
	return m
}

func (m *memory) Close() error {
	select {
	case <-m.exit:
		return nil
	default:
		close(m.exit)
	}
	return nil
}

func (m *memory) Load(sources ...source.Source) error {
	var errs []string
	for _, s := range sources {
		set, err := s.Read()
		if err != nil {
			errs = append(errs, fmt.Sprintf("error loading source %s: %v", s, err))
			continue
		}
		if set == nil {
			continue
		}
		m.Lock()
		m.sources = append(m.sources, s)
		m.sets = append(m.sets, set)
		idx := len(m.sets) - 1
		m.Unlock()
		go m.watch(idx, s)
	}

	if err := m.reload(); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (m *memory) Snapshot() (*loader.Snapshot, error) {
	if err := m.Sync(); err != nil {
		return nil, err
	}
	m.RLock()
	snap := loader.Copy(m.snap)
	m.RUnlock()
	return snap, nil
}

func (m *memory) Sync() error {
	var sets []*source.ChangeSet
	m.Lock()
	var errs []string
	for _, s := range m.sources {
		ch, err := s.Read()
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		sets = append(sets, ch)
	}
	m.sets = sets
	m.Unlock()

	if err := m.reload(); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("source loading errors: %s", strings.Join(errs, "\n"))
	}
	return nil
}

func (m *memory) Watch(keys ...string) (loader.Watcher, error) {
	value, err := m.get(keys...)
	if err != nil {
		return nil, err
	}
	w := &watcher{
		exit:   make(chan bool),
		key:    make([]string, 0),
		value:  value,
		reader: m.opts.Reader,
		data:   make(chan reader.Value, 1),
	}
	w.key = append(w.key, keys...)
	m.Lock()
	id := m.idx
	m.watchers[id] = w
	m.idx++
	m.Unlock()

	go func() {
		<-w.exit
		m.Lock()
		delete(m.watchers, id)
		m.Unlock()
	}()
	return w, nil
}

func (m *memory) Listen(v interface{}) loader.Refresher {
	switch cc := v.(type) {
	case loader.AutoLoader:
		return m.listen(cc)
	default:
		vv := &loader.Value{}
		vv.Value.Store(v) // 原始值
		vv.Tp = reflect.TypeOf(v)
		vv.Format = m.opts.Reader.String()
		return m.listen(vv)
	}
}

func (m *memory) listen(cc loader.AutoLoader) loader.Refresher {
	go func() {
		for {
			select {
			case <-m.lisChan:
				m.Lock()
				data := m.vals.Bytes()
				m.Unlock()
				if len(data) == 0 {
					continue
				}
				cc.Decode(data)
			}
		}
	}()
	return cc
}

func (m *memory) loaded() bool {
	var loaded bool
	m.RLock()
	if m.vals != nil {
		loaded = true
	}
	m.RUnlock()
	return loaded
}

func (m *memory) flush() error {
	m.Lock()
	defer m.Unlock()
	set, err := m.opts.Reader.Merge(m.sets...)
	if err != nil {
		return err
	}
	m.vals, err = m.opts.Reader.Values(set)
	if err != nil {
		return err
	}
	m.snap = &loader.Snapshot{
		ChangeSet: set,
		Version:   fmt.Sprintf("%d", time.Now().Unix()),
	}
	return nil
}

func (m *memory) reload() error {
	if err := m.flush(); err != nil {
		return err
	}
	m.notify()
	return nil
}

func (m *memory) notify() {
	var watchers []*watcher
	m.RLock()
	for _, w := range m.watchers {
		watchers = append(watchers, w)
	}
	m.RUnlock()

	for _, w := range watchers {
		m.RLock()
		data := m.vals.Get(w.key...)
		m.RUnlock()
		select {
		case w.data <- data:
		default:
		}
	}
}

func (m *memory) get(keys ...string) (reader.Value, error) {
	if !m.loaded() {
		if err := m.Sync(); err != nil {
			return nil, err
		}
	}
	m.Lock()
	defer m.Unlock()
	if m.vals != nil {
		return m.vals.Get(keys...), nil
	}
	ch := m.snap.ChangeSet
	v, err := m.opts.Reader.Values(ch)
	if err != nil {
		return nil, err
	}
	m.vals = v
	if m.vals != nil {
		return m.vals.Get(keys...), nil
	}

	return nil, errors.New("no values")
}

// fixme: 同一个配置项在不同的source里都配置了，从其中一个source删除后，该配置项还会存在
// 需要使用者保证同一个配置项只能配置在一个source中
// idx为加载的先后顺序
func (m *memory) watch(idx int, s source.Source) {
	watch := func(idx int, w source.Watcher) error {
		for {
			cs, err := w.Next()
			if err != nil {
				return err // watcher关闭
			}
			if cs == nil || len(cs.Data) == 0 {
				continue
			}
			m.Lock()
			m.sets[idx] = cs
			m.Unlock()
			if err := m.flush(); err != nil { // 解析错误
				logging.GenLogf("on memory watch, load value failed, err %v", err)
				continue
			}
			select {
			case m.lisChan <- true: // 更新listen的结构
			default:
			}
			m.notify() // 通知外部用户自定义的watcher
		}
	}

	for {
		// source Watch
		w, err := s.Watch()
		if err != nil {
			logging.GenLogf("make watcher failed, err %v, source: %+v", err, s)
			time.Sleep(time.Second)
			continue
		}

		done := make(chan bool)
		go func() {
			select {
			case <-done:
			case <-m.exit:
			}
			w.Stop()
		}()

		// block watch
		if err := watch(idx, w); err != nil {
			time.Sleep(2 * time.Second)
		}
		close(done)
		select {
		case <-m.exit:
			return
		default:
		}
	}
}

// 实现loader.Watcher接口
type watcher struct {
	exit   chan bool
	key    []string
	value  reader.Value
	reader reader.Reader
	data   chan reader.Value
}

func (w *watcher) Next() (*loader.Snapshot, error) {
	for {
		select {
		case <-w.exit:
			return nil, errors.New("watcher stopped")
		case v := <-w.data:
			if v == nil {
				continue
			}
			if bytes.Equal(w.value.Bytes(), v.Bytes()) {
				continue
			}
			w.value = v
			cs := &source.ChangeSet{
				Data:      v.Bytes(),
				Format:    w.reader.String(),
				Source:    "memory",
				Timestamp: time.Now(),
			}
			cs.Sum()
			return &loader.Snapshot{
				ChangeSet: cs,
				Version:   fmt.Sprintf("%d", time.Now().Unix()),
			}, nil
		}
	}
}

func (w *watcher) Stop() error {
	select {
	case <-w.exit:
	default:
		close(w.exit)
	}
	return nil
}
