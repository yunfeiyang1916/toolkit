package upstream

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
)

const (
	weightTagsKey = "__weight"
)

type Cluster struct {
	name                string
	checker             *HealthChecker
	detector            Detector
	registerBackend     registry.Backend
	hostMap             atomic.Value
	hostSet             *HostSet
	maxWeight           uint32
	conf                config.Cluster
	balancer            atomic.Value
	hostChangelist      atomic.Value
	hostChangedCallback atomic.Value
}

func NewCluster(conf config.Cluster, backend registry.Backend) *Cluster {
	conf = normalizeClusterConfig(conf)
	c := &Cluster{
		name:            conf.Name,
		registerBackend: backend,
		checker:         NewHealthChecker(TCP, time.Duration(conf.CheckInterval), conf.UnHealthyThreshold, conf.HealthyThreshold, conf.Name),
		hostSet:         NewHostSet(nil, nil),
		conf:            conf,
		maxWeight:       0,
	}
	hostMap := make(map[string]*Host)
	c.hostMap.Store(hostMap)
	//TODO dump last service lists
	c.detector = NewSimpleDector(c, conf.Detector)
	c.detector.AddChangedStateCallback(func(h *Host) {
		c.reloadHealthyHosts()
	})
	c.buildBalancer()
	c.hostChangelist.Store(changelist{})
	c.dealHostChanged()
	c.listenConfigChange()
	c.checker.AddHostCheckCompleteCb(func(h *Host, state HealthTransition) {
		if state == Changed {
			logging.Infof("service %s host %s stat changed to health=%t", c.name, h.Address(), h.Healthy())
			c.reloadHealthyHosts()
		}
	})
	return c
}

func normalizeClusterConfig(conf config.Cluster) config.Cluster {
	cfg := conf
	if time.Duration(cfg.CheckInterval) < time.Second {
		cfg.CheckInterval = config.Duration(5 * time.Second)
	}
	if cfg.LBPanicThreshold == 0 {
		cfg.LBPanicThreshold = 50
	}
	if cfg.UnHealthyThreshold < 1 {
		cfg.UnHealthyThreshold = 10
	}
	if cfg.HealthyThreshold < 1 {
		cfg.HealthyThreshold = 2
	}
	if time.Duration(cfg.Detector.DetectInterval) < time.Second {
		cfg.Detector.DetectInterval = config.Duration(20 * time.Second)
	}
	if time.Duration(cfg.Detector.BaseEjectionDuration) < time.Second {
		cfg.Detector.BaseEjectionDuration = config.Duration(100 * time.Millisecond)
	}
	if cfg.Detector.ConsecutiveError == 0 {
		cfg.Detector.ConsecutiveError = 100
	}
	if cfg.Detector.ConsecutiveConnectionError == 0 {
		cfg.Detector.ConsecutiveConnectionError = 5
	}
	if cfg.Detector.MaxEjectionPercent == 0 {
		cfg.Detector.MaxEjectionPercent = 50
	}
	if cfg.Detector.SuccessRateMinHosts == 0 {
		cfg.Detector.SuccessRateMinHosts = 3
	}
	if cfg.Detector.SuccessRateRequestVolume == 0 {
		cfg.Detector.SuccessRateRequestVolume = 100

	}
	if cfg.Detector.SuccessRateStdevFactor <= 1.0 {
		cfg.Detector.SuccessRateStdevFactor = 1900
	}
	return cfg
}

type changelist struct {
	current []*Host
	added   []*Host
	removed []*Host
	version uint32
}

func (c *Cluster) dealHostChanged() {
	go func() {
		var curIndex uint32
		for {
			info := c.hostChangelist.Load().(changelist)
			lastIndex := atomic.LoadUint32(&info.version)
			if lastIndex <= curIndex { // 无变动,忽略
				time.Sleep(10 * time.Millisecond)
				continue
			}
			curIndex = lastIndex
			hostAdded := make([]string, len(info.added))
			hostRemoved := make([]string, len(info.removed))
			hostCurrent := make([]string, len(info.current))
			for i, h := range info.added {
				hostAdded[i] = h.Address()
			}
			for i, h := range info.removed {
				hostRemoved[i] = h.Address()
			}
			for i, h := range info.added {
				hostCurrent[i] = h.Address()
			}
			if f, ok := c.hostChangedCallback.Load().(func([]string, []string, []string)); ok {
				f(hostCurrent, hostAdded, hostRemoved)
			}
		}
	}()
}

func (c *Cluster) AddHostChangedCallback(fn func(current []string, added []string, removed []string)) {
	if _, ok := c.hostChangedCallback.Load().(func([]string, []string, []string)); !ok {
		c.hostChangedCallback.Store(fn)
	}
}

func (c *Cluster) GetHostByAddress(address string) *Host {
	hostMap := c.hostMap.Load().(map[string]*Host)
	return hostMap[address]
}

func (c *Cluster) Close() {
}

func (c *Cluster) buildBalancer() {
	balanceType := UnmarshalBalanceFromText(c.conf.LBType)
	c.balancer.Store(NewSubsetBalancer(c.hostSet, balanceType, c.conf.LBPanicThreshold, c.conf.LBSubsetKeys, c.conf.LBDefaultKeys, c.conf.Name))
}

// HealthChecker ...
func (c *Cluster) HealthChecker() *HealthChecker {
	return c.checker
}

func (c *Cluster) reloadHealthyHosts() {
	current := make([]*Host, len(c.hostSet.Hosts()))
	copy(current, c.hostSet.Hosts())
	c.updateSet(current, nil, nil)
}

const (
	clustLastEndpointsFileDir   = "./data"
	clusterEndpointsFileNamePre = ".endpoints"
)

func getEndpointsDumpFileName(name string) string {
	return filepath.Join(clustLastEndpointsFileDir, fmt.Sprintf("%s.%s", clusterEndpointsFileNamePre, name))
}

func (c *Cluster) dumpLastServerResult(serverList *registry.Cluster) error {
	if len(serverList.Endpoints) > 0 {
		if _, err := os.Stat(clustLastEndpointsFileDir); err != nil {
			if err = os.Mkdir("./data", 0700); err != nil {
				logging.Warnf("dumpLastServerResult error %s", err)
				return err
			}
		}
	}
	serverList.AddEnvTag()
	tmpFile, err := ioutil.TempFile(clustLastEndpointsFileDir, ".endpoints.tmp")
	if err != nil {
		logging.Warnf("open endpoints tmp file error %s", err)
		return err
	}
	defer tmpFile.Close()
	err = json.NewEncoder(tmpFile).Encode(serverList)
	if err != nil {
		logging.Warnf("dump endpoints to tmp file error %s", err)
		return err
	}
	err = os.Rename(tmpFile.Name(), getEndpointsDumpFileName(c.name))
	logging.Infof("dumpLastServerResult %s return %v, last service list size %d", c.name, err, len(serverList.Endpoints))
	return err
}
func (c *Cluster) loadLastServiceResult() (registry.Cluster, error) {
	var cluster registry.Cluster
	content, err := ioutil.ReadFile(getEndpointsDumpFileName(c.name))
	if err != nil {
		return cluster, err
	}
	err = json.Unmarshal(content, &cluster)
	if err != nil {
		logging.Warnf("load last service result parse data error %s", err)
		return cluster, err
	}
	if cluster.Name != c.name {
		return cluster, fmt.Errorf("last service list name is not euqal specified name, file content name %s, cluster name %s", cluster.Name, c.name)
	}
	cluster.AddEnvTag()
	return cluster, err
}

func (c *Cluster) listenConfigChange() {
	ch := c.registerBackend.WatchServices(c.name, []string{"passing", "warnning"}, c.conf.Datacenter)
	var staticEndpoints []string
	if len(c.conf.StaticEndpoints) != 0 {
		staticEndpoints = strings.Split(c.conf.StaticEndpoints, ",")
	}
	var cs []*registry.Cluster
	select {
	case cs = <-ch:
	case <-time.After(500 * time.Millisecond):
		var one registry.Cluster
		var err error
		one, err = c.loadLastServiceResult()
		if err != nil {
			one = registry.Cluster{
				Name:      c.name,
				Endpoints: make([]registry.Endpoint, 0),
			}
			logging.Warnf("cluster %q use staticEndpoints %v, load last error %s", c.name, staticEndpoints, err)
			for i, end := range staticEndpoints {
				host, portStr, err := net.SplitHostPort(end)
				if err != nil {
					continue
				}
				port, _ := strconv.Atoi(portStr)
				one.Endpoints = append(one.Endpoints, registry.Endpoint{
					ID:   strconv.Itoa(i),
					Addr: host,
					Port: port,
					Tags: []string{"env=online"}, // 使用配置文件中指定值,默认添加env
				})

			}
		}
		cs = make([]*registry.Cluster, 1)
		cs[0] = &one
		logging.Warnf("cluster %q use startup endpoints %v", c.name, one.Endpoints)
	}
	if len(cs) < 1 {
		logging.Warnf("cluster %q has no services", c.name)
	}
	// on init stage, add env tag
	if len(cs) > 0 {
		cluster := cs[0]
		cluster.AddEnvTag()
		c.onClusterChanged(cluster)
	}
	go func() {
		for cs := range ch {
			if len(cs) < 1 || len(cs[0].Endpoints) < 1 {
				logging.Warnf("cluster %q active endpoints become size 0, ignore service list change", c.name)
				continue
			}
			newCluster := cs[0]
			c.onClusterChanged(newCluster)
		}
	}()
}

func (c *Cluster) Balancer() Balancer {
	return c.balancer.Load().(Balancer)
}

func (c *Cluster) onClusterChanged(newCluster *registry.Cluster) {
	// dumpLastServerResult ignore error
	c.dumpLastServerResult(newCluster)
	var (
		hostAdded   = []*Host{}
		hostRemoved = []*Host{}
		newHostsMap = map[string]*Host{}
		current     = make([]*Host, len(c.hostSet.Hosts()))
	)
	copy(current, c.hostSet.Hosts())
	newHosts := convertEndpointsToHosts(newCluster)
	c.updateDynamicHostList(newHosts, &hostAdded, &hostRemoved, &current, newHostsMap)
	c.updateSet(current, hostAdded, hostRemoved)
	c.hostMap.Store(newHostsMap)
	logging.Infof("%s onClusterChanged %v, added %v, removed %v, current %v", c.name, newCluster.Endpoints, hostsToString(hostAdded), hostsToString(hostRemoved), hostsToString(current))

	// 更新变更信息
	changed := c.hostChangelist.Load().(changelist)
	atomic.AddUint32(&changed.version, 1)
	changed.current = current
	changed.added = hostAdded
	changed.removed = hostRemoved
	c.hostChangelist.Store(changed)
}

func (c *Cluster) updateSet(current, added, removed []*Host) {
	health := make([]*Host, 0)
	for _, h := range current {
		if h.Healthy() {
			health = append(health, h)
		}
	}
	logging.Infof("%s host set updated, now health %v", c.name, hostsToString(health))
	c.hostSet.UpdateHosts(current, health, added, removed)
	c.checker.OnHostsChanged(added, removed)
}

func (c *Cluster) updateDynamicHostList(newHost []*Host, added, removed, current *[]*Host, updatedHosts map[string]*Host) bool {
	var (
		maxHostWeight uint32 = 1
		hostsChanged         = false
		finaHosts            = []*Host{}
		existingHosts        = map[string]*Host{}
	)
	hostMap := c.hostMap.Load().(map[string]*Host)
	for _, h := range newHost {
		if _, ok := updatedHosts[h.Address()]; ok {
			continue
		}
		existingHost, existing := hostMap[h.Address()]
		if existing {
			existingHosts[h.Address()] = h
			if h.Weight() > maxHostWeight {
				maxHostWeight = h.Weight()
			}
			if existingHost.HealthFlagGet(FailedRegistryHealth) != h.HealthFlagGet(FailedRegistryHealth) {
				previous := existingHost.Healthy()
				if h.HealthFlagGet(FailedRegistryHealth) {
					existingHost.HealthFlagSet(FailedRegistryHealth)
					hostsChanged = hostsChanged || previous
				} else {
					existingHost.HealthFlagClear(FailedRegistryHealth)
					hostsChanged = hostsChanged || (!previous && existingHost.Healthy())
				}
			}
			existingHost.SetWeight(h.Weight())
			existingHost.SetMeta(h.Meta())
			finaHosts = append(finaHosts, existingHost)
			updatedHosts[existingHost.Address()] = existingHost
		} else {
			if (h.Weight()) > maxHostWeight {
				maxHostWeight = h.Weight()
			}
			h.HealthFlagSet(FailedActiveHC)
			finaHosts = append(finaHosts, h)
			updatedHosts[h.Address()] = h
			*added = append(*added, h)
		}
	}
	for _, h := range *current {
		if _, ok := existingHosts[h.Address()]; !ok {
			*removed = append(*removed, h)
		}
	}
	*current = finaHosts
	atomic.StoreUint32(&c.maxWeight, maxHostWeight)
	return hostsChanged
}

func convertEndpointsToHosts(newCluster *registry.Cluster) []*Host {
	hosts := make([]*Host, 0, len(newCluster.Endpoints))
	for _, end := range newCluster.Endpoints {
		tags := make(map[string]string)
		var weight uint32 = 100
		for _, t := range end.Tags {
			kv := strings.Split(t, "=")
			if len(kv) == 2 {
				k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
				if k == weightTagsKey {
					weightV, err := strconv.Atoi(v)
					if err == nil && weightV >= 0 {
						weight = uint32(weightV)
					}
					continue
				}
				tags[k] = v
			}
		}
		h := NewHost(fmt.Sprintf("%s:%d", end.Addr, end.Port), weight, tags)
		hosts = append(hosts, h)
	}
	return hosts
}
