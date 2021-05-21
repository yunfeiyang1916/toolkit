package consul

import (
	"errors"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
	"github.com/yunfeiyang1916/toolkit/logging"
)

// be is an implementation of a registry backend for consul.
type be struct {
	c      *api.Client
	dc     string
	cfg    *config.Consul
	logger *logging.Logger
}

func NewBackend(cfg *config.Consul) (registry.Backend, error) {
	// create a reusable client
	c, err := api.NewClient(&api.Config{Address: cfg.Addr, Scheme: cfg.Scheme, Token: cfg.Token, WaitTime: 60 * time.Second})
	if err != nil {
		return nil, err
	}
	logger := cfg.Logger
	if logger == nil {
		logger = logging.New()
	}

	// ping the agent
	dc, err := datacenter(c)
	if err != nil {
		logger.Warnf("consul backend init error %s", err)
		// return nil, err
	}

	// we're good
	logger.Infof("consul: Connecting to %q in datacenter %q", cfg.Addr, dc)
	return &be{c: c, dc: dc, cfg: cfg, logger: logger}, nil
}

func (b *be) Register(cfg *config.Register) error {
	tagsValue := <-cfg.TagsOverrideCh
	service, err := serviceRegistration(cfg, tagsValue)
	if err != nil {
		return err
	}

	cfg.DerigesterCh = register(b.logger, b.c, service, cfg.TagsOverrideCh)
	return nil
}

func (b *be) Deregister(cfg *config.Register) error {
	cfg.DerigesterCh <- true // trigger deregistration
	<-cfg.DerigesterCh
	return nil
}

func (b *be) ReadManual(KVPath string) (value string, version uint64, err error) {
	// we cannot rely on the value provided by WatchManual() since
	// someone has to call that method first to kick off the go routine.
	return getKV(b.c, KVPath, 0)
}

func (b *be) WriteManual(KVPath, value string, version uint64) (ok bool, err error) {
	// try to create the key first by using version 0
	if ok, err = putKV(b.c, KVPath, value, 0); ok {
		return
	}

	// then try the CAS update
	return putKV(b.c, KVPath, value, version)
}

func (b *be) WatchServices(name string, status []string, dc string) chan []*registry.Cluster {
	b.logger.Infof("consul: Watching Services %q, status %v", name, status)

	svc := make(chan []*registry.Cluster)
	go watchServices(b.logger, b.c, name, status, dc, svc)
	return svc
}

func (b *be) WatchManual(KVPath string) chan string {
	b.logger.Infof("consul: Watching KV path %q", KVPath)
	kv := make(chan string)
	go watchKV(b.logger, b.c, KVPath, kv)
	return kv
}

func (b *be) WatchPrefixManual(prefix string) chan map[string]string {
	b.logger.Infof("consul: Watching prefix path %q", prefix)
	kvs := make(chan map[string]string)
	go watchPrefix(b.logger, b.c, prefix, kvs)
	return kvs
}

// datacenter returns the datacenter of the local agent
func datacenter(c *api.Client) (string, error) {
	self, err := c.Agent().Self()
	if err != nil {
		return "", err
	}

	cfg, ok := self["Config"]
	if !ok {
		return "", errors.New("consul: self.Config not found")
	}
	dc, ok := cfg["Datacenter"].(string)
	if !ok {
		return "", errors.New("consul: self.Datacenter not found")
	}
	return dc, nil
}
