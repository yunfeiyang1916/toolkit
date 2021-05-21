package upstream

import (
	"sort"
	"sync/atomic"
	"time"

	"math/rand"

	"github.com/pierrec/xxHash/xxHash64"
	"golang.org/x/net/context"
)

type BalanceType int32

const (
	RoundRobin BalanceType = iota
	LeastRequest
	Random
	RingHash
	WeightRoundRobin
	SubsetBalanceType
)

func (t BalanceType) String() string {
	switch t {
	case RoundRobin:
		return "RoundRobin"
	case LeastRequest:
		return "LeastRequest"
	case Random:
		return "Random"
	case RingHash:
		return "RingHash"
	case WeightRoundRobin:
		return "WeightRoundRobin"
	case SubsetBalanceType:
		return "Subset"
	}
	return "Unknown"
}

func UnmarshalBalanceFromText(t string) BalanceType {
	switch t {
	case "RoundRobin", "roundrobin", "round_robin":
		return WeightRoundRobin
	case "Random", "random":
		return Random
	case "RingHash", "hash":
		return RingHash
	case "WeightRoundRobin", "weight_roundrobin", "weight_round_robin":
		return WeightRoundRobin
	default:
		panic("Unknown Balance Type")
	}

}

type Balancer interface {
	ChooseHost(ctx context.Context) *Host
	Hosts(ctx context.Context) []*Host
}

func isGlobalPanic(set *HostSet, threshhold int) bool {
	hostSize := len(set.Hosts())
	helthSize := len(set.HelathHosts())
	healthyPercent := 0
	if hostSize != 0 {
		healthyPercent = int(100 * (float64(helthSize) / float64(hostSize)))
	}
	return healthyPercent < threshhold
}

type RoundRobinBalancer struct {
	panicThreshold int
	hostSet        *HostSet
	usedIndex      uint64
}

func NewRoundRobinBalancer(set *HostSet, threshhold int) *RoundRobinBalancer {
	return &RoundRobinBalancer{
		panicThreshold: threshhold,
		hostSet:        set,
		usedIndex:      uint64(rand.Int63()),
	}
}

func (rr *RoundRobinBalancer) ChooseHost(ctx context.Context) *Host {
	hosts := rr.hostSet.HelathHosts()
	if isGlobalPanic(rr.hostSet, rr.panicThreshold) {
		hosts = rr.hostSet.Hosts()
	}
	if len(hosts) == 0 {
		return nil
	}
	old := atomic.AddUint64(&rr.usedIndex, 1)
	return hosts[int(old)%len(hosts)]
}

func (rr *RoundRobinBalancer) Hosts(context.Context) []*Host {
	return rr.hostSet.Hosts()
}

type RandomBalancer struct {
	panicThreshold int
	hostSet        *HostSet
}

func NewRandomBalancer(set *HostSet, threshhold int) *RandomBalancer {
	return &RandomBalancer{
		panicThreshold: threshhold,
		hostSet:        set,
	}
}

func (rr *RandomBalancer) ChooseHost(ctx context.Context) *Host {
	hosts := rr.hostSet.HelathHosts()
	if isGlobalPanic(rr.hostSet, rr.panicThreshold) {
		hosts = rr.hostSet.Hosts()
	}
	if len(hosts) == 0 {
		return nil
	}
	return hosts[int(time.Now().UnixNano()/int64(time.Microsecond))%len(hosts)]
	// return hosts[rand.Intn(len(hosts))]
}
func (rr *RandomBalancer) Hosts(context.Context) []*Host {
	return rr.hostSet.Hosts()
}

type RingHashBalancer struct {
	hosts     []*Host
	tableSize uint64
	table     []*Host
}

type tableEntry struct {
	host   *Host
	offset uint64
	skip   uint64
	weight uint32
	counts uint64
	next   uint64
}

const tableSize uint64 = 65537

/**
* Maglev 一致hash算法
* https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/44824.pdf
**/
func NewRingHashBalancer(set *HostSet, threshhold int) *RingHashBalancer {
	rh := &RingHashBalancer{
		tableSize: tableSize,
	}
	if isGlobalPanic(set, threshhold) {
		rh.hosts = set.Hosts()

	} else {
		rh.hosts = set.HelathHosts()
	}
	rh.build()
	return rh
}

func (rh *RingHashBalancer) ChooseHost(ctx context.Context) *Host {
	hash, ok := HashFromContext(ctx)
	if !ok {
		hash = rand.Uint32()
	}
	return rh.table[int(uint64(hash)%uint64(rh.tableSize))]
}
func (rr *RingHashBalancer) Hosts(context.Context) []*Host {
	return rr.hosts
}

func (rh *RingHashBalancer) permutation(te *tableEntry) uint64 {
	return (te.offset + te.skip*te.next) % rh.tableSize
}

func (rh *RingHashBalancer) build() {
	var maxHostWeight uint32 = 0
	totalHosts := 0
	for _, h := range rh.hosts {
		if h.Weight() > maxHostWeight {
			maxHostWeight = h.Weight()
		}
		totalHosts++
	}
	if totalHosts == 0 {
		return
	}
	tableBuildEntry := make([]*tableEntry, 0, totalHosts)
	rh.table = make([]*Host, rh.tableSize)
	for _, h := range rh.hosts {
		address := []byte(h.Address())
		tableBuildEntry = append(
			tableBuildEntry,
			&tableEntry{
				host:   h,
				offset: xxHash64.Checksum(address, 0) % rh.tableSize,
				skip:   xxHash64.Checksum(address, 1)%(rh.tableSize-1) + 1,
				weight: h.Weight(),
			},
		)
	}
	var tableIndex uint64 = 0
	var iteration uint32 = 1
	for {
		for _, entry := range tableBuildEntry {
			if maxHostWeight > 0 {
				if uint64(iteration*entry.weight) < entry.counts {
					continue
				}
				entry.counts += uint64(maxHostWeight)
			}
			c := rh.permutation(entry)
			for rh.table[c] != nil {
				entry.next++
				c = rh.permutation(entry)
			}
			rh.table[c] = entry.host
			entry.next++
			tableIndex++
			if tableIndex == rh.tableSize {
				return
			}
		}
		iteration++
	}
}

type WeightRoundRobinBalancer struct {
	hosts       []*Host
	weightHosts []*Host
	weights     []float64
	usedIndex   uint64
}

const maxSlots = 1e4

func NewWeightRoundRobinBalancer(set *HostSet, threshhold int) *WeightRoundRobinBalancer {
	wrr := &WeightRoundRobinBalancer{
		usedIndex: uint64(rand.Int63()),
	}
	if isGlobalPanic(set, threshhold) {
		wrr.hosts = set.Hosts()

	} else {
		wrr.hosts = set.HelathHosts()
	}
	wrr.build()
	return wrr
}

func (wrr *WeightRoundRobinBalancer) ChooseHost(ctx context.Context) *Host {
	if len(wrr.weightHosts) == 0 {
		return nil
	}
	index := atomic.AddUint64(&wrr.usedIndex, 1)
	return wrr.weightHosts[int(index)%len(wrr.weightHosts)]
}

func (wrr *WeightRoundRobinBalancer) build() {
	// 计算有权重的host数量
	var nFixed int
	var sumFixed float64
	for _, h := range wrr.hosts {
		if h.Weight() > 0 {
			nFixed++
			sumFixed += float64(h.Weight()) / 100.0
		}
	}
	// 如果所有机器都没有配置权重，权重为1/n
	wrr.weights = make([]float64, len(wrr.hosts))
	if nFixed == 0 {
		w := 1.0 / float64(len(wrr.hosts))
		for index, _ := range wrr.hosts {
			wrr.weights[index] = w
		}
		wrr.weightHosts = wrr.hosts
		return
	}
	// 归一化权重
	scale := 1.0
	if sumFixed > 1 || (nFixed == len(wrr.hosts) && sumFixed < 1) {
		scale = 1 / sumFixed
	}
	// 补全没有权重的机器
	dynamic := (1 - sumFixed) / float64(len(wrr.hosts)-nFixed)
	if dynamic < 0 {
		dynamic = 0
	}
	// 计算每个host的真实权重
	for index, h := range wrr.hosts {
		if h.Weight() > 0 {
			wrr.weights[index] = float64(h.Weight()) / 100.0 * scale
		} else {
			wrr.weights[index] = dynamic
		}
	}
	// 计算round-robin分布
	// 1. 计算需要的round-robin环的数量，例如如果两个host为50%，50%环的数量为2， 如果10%-90%环的数量为10
	// 使用10000个slot，权重可以精确到0.01%
	// 2. 把给定的host分散到round-robin环上
	slots := make(byN, len(wrr.hosts))
	usedSlots := 0
	for i := range wrr.hosts {
		n := int(float64(maxSlots) * wrr.weights[i])
		if n == 0 && wrr.weights[i] > 0 {
			n = 1
		}
		slots[i].i = i
		slots[i].n = n
		usedSlots += n

	}
	sort.Sort(slots)
	targets := make([]*Host, usedSlots)
	for _, s := range slots {
		if s.n <= 0 {
			continue
		}
		next, step := 0, usedSlots/s.n
		for k := 0; k < s.n; k++ {
			for targets[next] != nil {
				next = (next + 1) % usedSlots
			}
			targets[next] = wrr.hosts[s.i]
			next = (next + step) % usedSlots
		}
	}
	wrr.weightHosts = targets
}

func (rr *WeightRoundRobinBalancer) Hosts(context.Context) []*Host {
	return rr.hosts
}

type byN []struct{ i, n int }

func (r byN) Len() int           { return len(r) }
func (r byN) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r byN) Less(i, j int) bool { return r[i].n < r[j].n }
