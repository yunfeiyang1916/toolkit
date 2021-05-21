package config

import (
	"bytes"
	"reflect"
	"strings"

	"github.com/yunfeiyang1916/toolkit/framework/config/reader"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
)

type Watcher interface {
	Next() map[string]string
}

// RemoteKV wrap Default's RemoteKV func, get kv string by remote path
func RemoteKV(p string) (string, error) {
	value, _, err := registry.Default.ReadManual(p)
	return value, err
}

// WatchPath wrap Default's WatchPath func, use for watching remote path data which maybe change anytime
func WatchKV(p string, prefix ...string) Watcher {
	w := &pathWatcher{
		path:  p,
		value: map[string]string{},
	}
	if len(prefix) > 0 {
		w.prefix = prefix[0]
	}
	// return string
	w.watchChan = registry.Default.WatchManual(p)
	return w
}

// WatchPrefix wrap Default's WatchPath func, use for watching remote path data which maybe change anytime
func WatchPrefix(p string) Watcher {
	w := &prefixWatcher{
		prefix: p,
		value:  map[string]string{},
	}
	// return map[string]string: key(is path) = value
	w.watchChan = registry.Default.WatchPrefixManual(p)
	return w
}

// Parse parse value to a struct
func Parse(data string, format string, structPtr interface{}) error {
	buf := bytes.NewBufferString(data)
	enc := reader.Encoder(format)
	return enc.Decode(buf.Bytes(), structPtr)
}

type prefixWatcher struct {
	watchChan chan map[string]string
	prefix    string
	value     map[string]string
}

func (p *prefixWatcher) Next() map[string]string {
	data := <-p.watchChan
	// no data
	if len(data) == 0 {
		return map[string]string{}
	}
	value := map[string]string{}
	for k, v := range data {
		kk := strings.TrimPrefix(k, strings.TrimPrefix(p.prefix, "/"))
		if len(v) == 0 {
			continue
		}
		value[kk] = v
	}
	// no changes
	if reflect.DeepEqual(p.value, value) {
		return map[string]string{}
	}
	p.value = value
	return p.value
}

type pathWatcher struct {
	watchChan chan string
	prefix    string
	path      string
	value     map[string]string
}

func (p *pathWatcher) Next() map[string]string {
	data := <-p.watchChan
	// no data
	if len(data) == 0 {
		return map[string]string{}
	}
	value := map[string]string{}
	if len(p.prefix) > 0 {
		key := strings.TrimPrefix(p.path, p.prefix)
		value[key] = data
	} else {
		value[p.path] = data
	}
	// no changes
	if reflect.DeepEqual(p.value, value) {
		return map[string]string{}
	}
	p.value = value
	return p.value
}
