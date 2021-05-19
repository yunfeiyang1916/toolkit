package gid

import (
	"math/rand"
	"strconv"
	"sync"
	"time"
)

var (
	g *gid
	h uint32
)

func init() {
	g = NewGId()
	h = HostnameHashCode()
}

type gid struct {
	startTime      int64
	durationBitNum uint32
	ipBitNum       uint32
	randBitNum     uint32
	pool           sync.Pool
}

func NewGId() *gid {
	seedGenerator := NewRand(time.Now().UnixNano())
	return &gid{
		startTime:      1546272000, // 从2019-01-01 00:00:00开始计算
		durationBitNum: 28,         // 时间间隔占用28bit
		ipBitNum:       16,         // IP占用16bit
		randBitNum:     20,         // 随机数占用20bit
		pool: sync.Pool{
			New: func() interface{} {
				return rand.NewSource(seedGenerator.Int63())
			},
		},
	}
}

func (g *gid) Rand() uint64 {
	number := g.RandNumber()
	duration := uint64(time.Now().Unix() - g.startTime)
	return duration<<g.durationBitNum | number>>g.durationBitNum
}

func (g *gid) RandNumber() uint64 {
	generator := g.pool.Get().(rand.Source)
	number := uint64(generator.Int63())
	g.pool.Put(generator)
	return number
}
func (g *gid) NewV1() uint64 {
	number := g.RandNumber()
	duration := uint64(time.Now().Unix() - g.startTime)
	ipCode := uint64(h % (1 << g.ipBitNum))
	result := duration<<(g.ipBitNum+g.randBitNum) | ipCode<<g.randBitNum | (number & ((1 << g.randBitNum) - 1))
	return result
}
func New() uint64 {
	return g.NewV1()
}
func (g *gid) UnixFromStr(s string) int64 {
	v, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return 0
	}
	return int64(v>>(g.ipBitNum+g.randBitNum)) + g.startTime
}
func UnixFromStr(s string) int64 {
	return g.UnixFromStr(s)

}
func (g *gid) UnixFromUint64(v uint64) int64 {
	return int64(v>>(g.ipBitNum+g.randBitNum)) + g.startTime
}
func UnixFromUint64(v uint64) int64 {
	return g.UnixFromUint64(v)
}
func (g *gid) StrToUint64(s string) uint64 {
	v, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return 0
	}
	return v
}
func StrToUint64(s string) uint64 {
	return g.StrToUint64(s)
}
func (g *gid) FnvCodeFromStr(s string) uint32 {
	v := g.StrToUint64(s)
	return uint32((v << g.durationBitNum) >> (g.durationBitNum + g.randBitNum))
}
func FnvCodeFromStr(s string) uint32 {
	return g.FnvCodeFromStr(s)
}
func (g *gid) FnvCodeFromUint64(v uint64) uint32 {
	return uint32((v << g.durationBitNum) >> (g.durationBitNum + g.randBitNum))
}
func FnvCodeFromUint64(v uint64) uint32 {
	return g.FnvCodeFromUint64(v)
}
func (g *gid) RandomFromUint64(v uint64) uint32 {
	return uint32(v & ((1 << g.randBitNum) - 1))
}
func RandomFromUint64(v uint64) uint32 {
	return g.RandomFromUint64(v)
}
func (g *gid) RandomFromStr(s string) uint32 {
	v := g.StrToUint64(s)
	return g.RandomFromUint64(v)
}
func RandomFromStr(s string) uint32 {
	return g.RandomFromStr(s)
}
