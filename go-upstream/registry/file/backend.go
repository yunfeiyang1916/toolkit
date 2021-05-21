// Package file implements a simple file based registry
// backend which reads the routes from a file once.
package file

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
)

type be struct {
	path       string
	lastData   *registry.Cluster
	lastModify time.Time
}

func NewBackend(filename string, initData *registry.Cluster) (registry.Backend, error) {
	return &be{path: filename, lastModify: time.Time{}, lastData: initData}, nil
}

func readFileData(path string) (*registry.Cluster, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var clusters registry.Cluster
	err = json.Unmarshal(bytes.TrimSpace(data), &clusters)
	if err != nil {
		return nil, err
	}
	return &clusters, nil
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
	if b.lastData != nil && len(b.lastData.Endpoints) != 0 {
		b.lastData.AddEnvTag()
		ch <- []*registry.Cluster{
			b.lastData,
		}
	}
	go func() {
		for {
			time.Sleep(10 * time.Second)
			st, err := os.Stat(b.path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[balance relaod] open file (path %s) error %s\n", b.path, err)
				continue
			}
			if st.ModTime().After(b.lastModify) {
				clusters, err := readFileData(b.path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[balance reload] reaload file (path %s) error %s\n", b.path, err)
					continue
				}
				fmt.Fprintf(os.Stdout, "[balance reload] reload file (path %s) success with data %+v\n", b.path, clusters)
				if len(clusters.Endpoints) == 0 {
					fmt.Fprintf(os.Stderr, "[balance reload] reload file (path %s) success with data %+v, size 0\n", b.path, clusters)
					continue
				}
				b.lastModify = st.ModTime()
				b.lastData = clusters
				b.lastData.AddEnvTag()
				ch <- []*registry.Cluster{
					b.lastData,
				}
			}
		}
	}()
	return ch
}

func (b *be) WatchManual(KVPath string) chan string {
	return make(chan string)
}

func (b *be) WatchPrefixManual(KVPath string) chan map[string]string {
	return make(chan map[string]string)
}
