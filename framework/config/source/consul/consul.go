package consul

import (
	"bytes"
	"time"

	"github.com/yunfeiyang1916/toolkit/logging"

	"github.com/yunfeiyang1916/toolkit/framework/config/source"
	clusterconfig "github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
	iconsul "github.com/yunfeiyang1916/toolkit/go-upstream/registry/consul"
)

// Currently a single consul reader
type consul struct {
	prefix      string
	stripPrefix string
	opts        source.Options
}

var (
	DefaultPrefix = "/service_config"
)

// read content by abs path
func (c *consul) Read() (*source.ChangeSet, error) {
	value, _, err := registry.Default.ReadManual(c.prefix)
	if err != nil {
		logging.GenLogf("consul ReadManual failed, key %s, err %v", c.prefix, err)
		return nil, err
	}
	if len(value) == 0 {
		logging.GenLogf("consul ReadManual empty value, key: %s", c.prefix)
		return &source.ChangeSet{}, nil
	}
	buf := bytes.NewBufferString(value)
	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Format:    c.opts.Encoder.String(),
		Source:    c.String(),
		Data:      buf.Bytes(),
	}
	cs.Checksum = cs.Sum()
	return cs, nil
}

func (c *consul) String() string {
	return "consul"
}

func (c *consul) Watch() (source.Watcher, error) {
	w := newWatcher(c.prefix, c.String(), c.stripPrefix, c.opts.Encoder)
	return w, nil
}

func NewSource(opts ...source.Option) source.Source {
	options := source.NewOptions(opts...)
	prefix := DefaultPrefix
	sp := ""
	f, ok := options.Context.Value(prefixKey{}).(string)
	if ok {
		prefix = f
	}
	if b, ok := options.Context.Value(stripPrefixKey{}).(bool); ok && b {
		sp = prefix
	}

	addr, _ := options.Context.Value(addressKey{}).(string)
	if len(addr) == 0 {
		addr = "127.0.0.1:8500"
	}
	if registry.Default == nil {
		registry.Default, _ = iconsul.NewBackend(&clusterconfig.Consul{Addr: addr, Scheme: "http", Logger: logging.Log(logging.GenLoggerName)})
	}

	return &consul{
		prefix:      prefix,
		stripPrefix: sp,
		opts:        options,
	}
}
