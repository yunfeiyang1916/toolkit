package upstream

import (
	"bytes"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"golang.org/x/net/context"
)

type subsetMetaData struct {
	key   string
	value string
}

type subsetMetaDatas []subsetMetaData

func (m subsetMetaDatas) Len() int {
	return len(m)
}

func (m subsetMetaDatas) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m subsetMetaDatas) Less(i, j int) bool {
	return m[i].key < m[j].key
}

type valueSubsetMap map[string]*subsetEntry

type subsetMap map[string]valueSubsetMap

type hostSubset struct {
	subsetLB  *SubsetBalancer
	subset    *HostSet
	lb        atomic.Value
	empty     bool
	predicate func(*Host) bool
	desc      string
}

type subsetEntry struct {
	children map[string]valueSubsetMap
	subset   *hostSubset
}

func (se *subsetEntry) initialized() bool {
	return se.subset != nil
}

type SubsetBalancer struct {
	hostSet           *HostSet
	subsetKeys        [][]string
	LBType            BalanceType
	LBPanicThreshold  int
	subsets           subsetMap
	fallbackSubset    *subsetEntry
	defaultSubsetMeta []subsetMetaData
	mu                *sync.RWMutex
	name              string
}

func NewSubsetBalancer(hostSet *HostSet, lbType BalanceType, threshhold int, keys [][]string, defaultKVs []string, name string) *SubsetBalancer {
	ss := &SubsetBalancer{
		hostSet:           hostSet,
		subsetKeys:        keys,
		LBType:            lbType,
		LBPanicThreshold:  threshhold,
		subsets:           make(subsetMap),
		fallbackSubset:    nil,
		defaultSubsetMeta: make([]subsetMetaData, 0),
		mu:                new(sync.RWMutex),
		name:              name,
	}
	ss.defaultSubsetMeta = stringSliceToMetadataSlice(defaultKVs)
	ss.refreshSubsets()
	ss.hostSet.AddUpdateCallback(func(added, removed []*Host) {
		if len(added) != 0 && len(removed) != 0 {
			logging.Debugf("%s hostset update added %v, removed %v, refresh", ss.name, added, removed)
			ss.refreshSubsets()
		} else {
			logging.Debugf("%s hostset update added %v, removed %v, update", ss.name, added, removed)
			ss.update(added, removed)
		}

	})
	return ss
}

func (ss *SubsetBalancer) Hosts(ctx context.Context) []*Host {
	entry := ss.chooseSubset(ctx)
	if entry != nil {
		return entry.subset.lb.Load().(Balancer).Hosts(ctx)
	}
	if ss.fallbackSubset == nil {
		return nil
	}
	return ss.fallbackSubset.subset.lb.Load().(Balancer).Hosts(ctx)
}

func (ss *SubsetBalancer) ChooseHost(ctx context.Context) *Host {
	h := ss.tryChooseHost(ctx)
	if h != nil {
		return h
	}
	if ss.fallbackSubset == nil {
		return nil
	}
	return ss.fallbackSubset.subset.lb.Load().(Balancer).ChooseHost(ctx)
}

func (ss *SubsetBalancer) tryChooseHost(ctx context.Context) *Host {
	entry := ss.chooseSubset(ctx)
	if entry == nil {
		return nil
	}
	return entry.subset.lb.Load().(Balancer).ChooseHost(ctx)
}

func (ss *SubsetBalancer) chooseSubset(ctx context.Context) *subsetEntry {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	carrier := extractSubsetCarrier(ctx)
	carrierSize := len(carrier)
	subsets := ss.subsets
	for i, kv := range carrier {
		subset, ok := subsets[kv.key]
		if !ok {
			break
		}
		entry, ok := subset[kv.value]
		if !ok {
			break
		}
		if i+1 == carrierSize {
			return entry

		}
		subsets = entry.children

	}
	return nil
}

func (ss *SubsetBalancer) refreshSubsets() {
	ss.update(ss.hostSet.Hosts(), nil)
}

func (ss *SubsetBalancer) update(added, removed []*Host) {
	ss.mu.Lock()
	ss.updateFallbackSubsets(added, removed)
	ss.processSubsets(added, removed)
	ss.mu.Unlock()
}

func (ss *SubsetBalancer) updateFallbackSubsets(added, removed []*Host) {
	if ss.fallbackSubset == nil {
		ss.fallbackSubset = new(subsetEntry)
		ss.fallbackSubset.subset = newHostSubset(ss, func(host *Host) bool {
			return hostMatches(ss.defaultSubsetMeta, host)
		}, describeMetadata(ss.name, ss.defaultSubsetMeta))
		logging.Debugf("%s subset lb: creating fallback load balancer for %s", ss.name, describeMetadata(ss.name, ss.defaultSubsetMeta))
	}
	ss.fallbackSubset.subset.update(added, removed)
}

func (ss *SubsetBalancer) processSubsets(added, removed []*Host) {
	subsetsModified := make(map[*subsetEntry]struct{})
	steps := []struct {
		hosts  []*Host
		adding bool
	}{{added, true}, {removed, false}}
	for _, step := range steps {
		hosts := step.hosts
		addingHosts := step.adding
		for _, h := range hosts {
			for _, keys := range ss.subsetKeys {
				kvs := extractSubsetMetadata(keys, h)
				if len(kvs) > 0 {
					entry := ss.findOrCreateSubsets(ss.subsets, kvs, 0, addingHosts)
					if entry == nil {
						continue
					}
					if _, ok := subsetsModified[entry]; ok {
						continue
					}
					subsetsModified[entry] = struct{}{}
					if entry.initialized() {
						logging.Debugf("%s subset lb: update load balancer for %s", ss.name, describeMetadata(ss.name, ss.defaultSubsetMeta))
						// update
						entry.subset.update(added, removed)
					} else {
						if addingHosts {
							// TODO new
							entry.subset = newHostSubset(ss, func(host *Host) bool { return hostMatches(kvs, host) }, describeMetadata(ss.name, kvs))
							logging.Infof("%s subset lb: creating load balancer for %s", ss.name, describeMetadata(ss.name, kvs))
						}
					}
				}
			}
		}
	}
	ss.forEachSubset(ss.subsets, func(entry *subsetEntry) {
		if _, ok := subsetsModified[entry]; ok {
			return
		}
		if entry.initialized() {
			// update
			entry.subset.update(added, removed)
		}
	})
}

func (ss *SubsetBalancer) forEachSubset(subsets subsetMap, cb func(*subsetEntry)) {
	for _, vsm := range subsets {
		for _, em := range vsm {
			entry := em
			cb(entry)
			ss.forEachSubset(entry.children, cb)
		}
	}
}

func (ss *SubsetBalancer) findOrCreateSubsets(subsets subsetMap, kvs []subsetMetaData, idx int, isAdding bool) *subsetEntry {
	var (
		name  = kvs[idx].key
		value = kvs[idx].value
		entry *subsetEntry
	)
	subset, ok := subsets[name]
	if ok {
		entry = subset[value]
	}
	if entry == nil {
		if !isAdding {
			return nil
		}
		entry = &subsetEntry{
			children: make(map[string]valueSubsetMap),
			subset:   nil,
		}
		if ok {
			subset[value] = entry
		} else {
			subsets[name] = valueSubsetMap{value: entry}
		}
	}
	idx++
	if idx == len(kvs) {
		return entry
	}
	return ss.findOrCreateSubsets(entry.children, kvs, idx, isAdding)
}

func newHostSubset(subsetLB *SubsetBalancer, predicate func(entry *Host) bool, desc string) *hostSubset {
	hs := &hostSubset{
		subsetLB:  subsetLB,
		subset:    NewHostSet(nil, nil),
		empty:     len(subsetLB.hostSet.Hosts()) == 0,
		predicate: predicate,
		desc:      desc,
	}
	hs.update(subsetLB.hostSet.Hosts(), nil)
	switch subsetLB.LBType {
	case RoundRobin:
		hs.lb.Store(NewRoundRobinBalancer(hs.subset, hs.subsetLB.LBPanicThreshold))
	case Random:
		hs.lb.Store(NewRandomBalancer(hs.subset, hs.subsetLB.LBPanicThreshold))
	case RingHash:
		hs.lb.Store(NewRingHashBalancer(hs.subset, hs.subsetLB.LBPanicThreshold))
		hs.subset.AddUpdateCallback(func(added, removed []*Host) {
			hs.lb.Store(NewRingHashBalancer(hs.subset, hs.subsetLB.LBPanicThreshold))

		})
	case WeightRoundRobin:
		hs.lb.Store(NewWeightRoundRobinBalancer(hs.subset, hs.subsetLB.LBPanicThreshold))
		hs.subset.AddUpdateCallback(func(added, removed []*Host) {
			logging.Debugf("%s subset weightRoundRobin update %v, removed %v", hs.desc, hostsToString(added), hostsToString(removed))
			hs.lb.Store(NewWeightRoundRobinBalancer(hs.subset, hs.subsetLB.LBPanicThreshold))
		})
	}
	return hs
}

func (hs *hostSubset) update(added, removed []*Host) {
	hostAdded := make(map[*Host]bool)
	filterAdded := make([]*Host, 0)
	for _, h := range added {
		if hs.predicate(h) {
			hostAdded[h] = true
			filterAdded = append(filterAdded, h)
		}
	}
	filterRemoved := make([]*Host, 0)
	for _, h := range removed {
		if hs.predicate(h) {
			filterRemoved = append(filterRemoved, h)
		}
	}
	hosts := make([]*Host, 0)
	healthHosts := make([]*Host, 0)
	for _, h := range hs.subsetLB.hostSet.Hosts() {
		_, hostSeen := hostAdded[h]
		if hostSeen || hs.predicate(h) {
			hosts = append(hosts, h)
			if h.Healthy() {
				healthHosts = append(healthHosts, h)
			}
		}
	}
	logging.Debugf("subset:%s origin hosts %v, hosts %v, healthy %v, filterAdded %v, filterRemoved %v", hs.desc, hostsToString(hs.subsetLB.hostSet.Hosts()), hostsToString(hosts), hostsToString(healthHosts), hostsToString(filterAdded), hostsToString(filterRemoved))
	hs.subset.UpdateHosts(hosts, healthHosts, filterAdded, filterRemoved)
	hs.empty = len(hs.subset.Hosts()) == 0
}

func hostMatches(kvs []subsetMetaData, host *Host) bool {
	hostMeta := host.Meta()
	for _, kv := range kvs {
		v, ok := hostMeta[kv.key]
		if !ok {
			return false
		}
		if v != kv.value {
			return false
		}
	}
	return true
}

func extractSubsetMetadata(keys []string, hosts *Host) []subsetMetaData {
	kvs := make([]subsetMetaData, 0)
	hostMeta := hosts.Meta()
	for _, k := range keys {
		if v, ok := hostMeta[k]; ok {
			kvs = append(kvs, subsetMetaData{
				key:   k,
				value: v,
			})
		}
	}
	if len(kvs) != len(keys) {
		return nil
	}
	sort.Sort(subsetMetaDatas(kvs))
	return kvs
}

func describeMetadata(name string, kvs []subsetMetaData) string {
	if len(kvs) == 0 {
		return name + ":<>"
	}
	buf := bytes.NewBuffer(nil)
	buf.WriteString(name)
	buf.WriteString(":")
	buf.WriteString("<")
	first := true
	for _, kv := range kvs {
		if !first {
			buf.WriteString(",")
		} else {
			first = false
		}
		fmt.Fprintf(buf, "%s=%s", kv.key, kv.value)
	}
	buf.WriteString(">")
	return buf.String()
}

type subsetCarrierKeyType struct{}

var (
	subsetCarrierKey = subsetCarrierKeyType{}
)

func extractSubsetCarrier(ctx context.Context) []subsetMetaData {
	carrier := ctx.Value(subsetCarrierKey)
	if carrier == nil {
		return nil
	}
	return carrier.([]subsetMetaData)
}

func InjectSubsetCarrier(ctx context.Context, kvs []string) context.Context {
	return context.WithValue(ctx, subsetCarrierKey, stringSliceToMetadataSlice(kvs))
}

func stringSliceToMetadataSlice(kvs []string) []subsetMetaData {
	data := make([]subsetMetaData, 0)
	if len(kvs)%2 == 0 {
		for i := 0; i < len(kvs)/2; i++ {
			data = append(data,
				subsetMetaData{
					key:   kvs[2*i],
					value: kvs[2*i+1],
				})
		}

	}
	sort.Sort(subsetMetaDatas(data))
	return data
}

func hostsToString(hs []*Host) string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("[")
	for _, h := range hs {
		fmt.Fprintf(buf, "{Address:%s, Tags:%v}", h.Address(), h.Meta())
	}
	buf.WriteString("]")
	return buf.String()
}
