package upstream

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry/file"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry/static"
	"golang.org/x/net/context"
)

const (
	filePrefix = "file://"
)

type ClusterManager struct {
	mutex    *sync.RWMutex
	clusters map[string]*Cluster
}

func NewClusterManager() *ClusterManager {
	return &ClusterManager{
		mutex:    new(sync.RWMutex),
		clusters: make(map[string]*Cluster),
	}
}

func (cm *ClusterManager) Cluster(name string) *Cluster {
	cm.mutex.Lock()
	c := cm.clusters[name]
	cm.mutex.Unlock()
	return c
}

func (cm *ClusterManager) InitService(conf config.Cluster) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	_, ok := cm.clusters[conf.Name]
	if ok {
		return nil
	}
	// if backend
	var backend registry.Backend
	if conf.EndpointsFrom != "consul" {
		clusterEndpoints := make([]registry.Endpoint, 0)
		if len(conf.StaticEndpoints) != 0 {
			for _, end := range strings.Split(conf.StaticEndpoints, ",") {
				host, port, err := net.SplitHostPort(strings.TrimSpace(end))
				if err != nil {
					return fmt.Errorf("init service %q error %s", conf.Name, err)
				}
				portNum, err := strconv.Atoi(port)
				if err != nil {
					return fmt.Errorf("%q endpoint port format error %s1", end, err)
				}
				clusterEndpoints = append(clusterEndpoints, registry.Endpoint{
					ID:   "",
					Addr: host,
					Port: portNum,
					Tags: []string{"env=online"}, // 静态方式,默认添加env
				})
			}
		}
		clusterHosts := make([]*registry.Cluster, 1)
		clusterHosts[0] = &registry.Cluster{
			Name:      conf.Name,
			Endpoints: clusterEndpoints,
		}
		if strings.HasPrefix(conf.EndpointsFrom, filePrefix) {
			backend, _ = file.NewBackend(strings.TrimPrefix(conf.EndpointsFrom, filePrefix), clusterHosts[0])
		} else {
			routers, _ := json.Marshal(clusterHosts)
			backend, _ = static.NewBackend(string(routers))
		}
	} else {
		backend = registry.Default
	}
	cluster := NewCluster(conf, backend)
	cm.clusters[conf.Name] = cluster
	return nil
}

func (cm *ClusterManager) ChooseHost(ctx context.Context, service string) *Host {
	cm.mutex.RLock()
	cluster, ok := cm.clusters[service]
	cm.mutex.RUnlock()
	if !ok {
		return nil
	}
	return cluster.Balancer().ChooseHost(ctx)
}

func (cm *ClusterManager) ChooseAllHosts(ctx context.Context, service string) []*Host {
	cm.mutex.RLock()
	cluster, ok := cm.clusters[service]
	cm.mutex.RUnlock()
	if !ok {
		return nil
	}
	return cluster.Balancer().Hosts(ctx)
}

func (cm *ClusterManager) PutResult(service, address string, code int) {
	cm.mutex.RLock()
	cluster, ok := cm.clusters[service]
	cm.mutex.RUnlock()
	if !ok {
		return
	}
	host := cluster.GetHostByAddress(address)
	if host == nil {
		return
	}
	host.GetDetectorMonitor().PutResult(Result(code))
}
