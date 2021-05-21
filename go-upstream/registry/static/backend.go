// Package static implements a simple static registry
// backend which uses statically configured routes.
package static

import (
	"encoding/json"

	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
)

type be struct {
	clusters []*registry.Cluster
}

func NewBackend(routes string) (registry.Backend, error) {
	b := be{}
	err := json.Unmarshal([]byte(routes), &(b.clusters))
	return &b, err
}

func (b *be) Register(*config.Register) error {
	return nil
}

func (b *be) Deregister(*config.Register) error {
	return nil
}

func (b *be) OverrideTags([]string) {
}

func (b *be) ReadManual(KVPath string) (value string, version uint64, err error) {
	return "", 0, nil
}

func (b *be) WriteManual(KVPath, value string, version uint64) (ok bool, err error) {
	return false, nil
}

// func (b *be) WatchServices() chan string {
func (b *be) WatchServices(name string, status []string, dc string) chan []*registry.Cluster {
	ch := make(chan []*registry.Cluster, 1)
	ch <- b.clusters
	return ch
}

func (b *be) WatchManual(KVPath string) chan string {
	return make(chan string)
}

func (b *be) WatchPrefixManual(KVPath string) chan map[string]string {
	return make(chan map[string]string)
}
