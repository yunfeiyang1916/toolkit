package upstream

import (
	"math/bits"
	"sync/atomic"
)

type Host struct {
	address                 string
	healthFlag              [4]uint32
	weight                  uint32
	used                    uint32
	activeHealthFailureType uint32
	meta                    atomic.Value
	detector                atomic.Value
}

func NewHost(address string, weight uint32, meta map[string]string) *Host {
	h := &Host{
		address:    address,
		used:       1,
		weight:     weight,
		healthFlag: [4]uint32{0},
	}
	h.meta.Store(meta)
	return h
}

func (h *Host) Address() string {
	return h.address
}

func (h *Host) Meta() map[string]string {
	return h.meta.Load().(map[string]string)
}

func (h *Host) SetMeta(meta map[string]string) {
	h.meta.Store(meta)
	return
}

func (h *Host) HealthFlagClear(flag HealthFlag) {
	atomic.StoreUint32(&h.healthFlag[bits.TrailingZeros32(uint32(flag))], 0)

}
func (h *Host) HealthFlagGet(flag HealthFlag) (res bool) {
	res = atomic.LoadUint32(&h.healthFlag[bits.TrailingZeros32(uint32(flag))]) == 1
	return

}

func (h *Host) HealthFlagSet(flag HealthFlag) {
	atomic.StoreUint32(&h.healthFlag[bits.TrailingZeros32(uint32(flag))], 1)
}

func (h *Host) Healthy() (health bool) {
	for idx := range h.healthFlag {
		if atomic.LoadUint32(&h.healthFlag[idx]) != 0 {
			return false
		}
	}
	return true

}
func (h *Host) GetActiveHealthFailureType() (tp ActiveHalthFailureType) {
	tp = ActiveHalthFailureType(atomic.LoadUint32(&h.activeHealthFailureType))
	return

}
func (h *Host) SetActiveHealthFailureType(tp ActiveHalthFailureType) {
	atomic.StoreUint32(&h.activeHealthFailureType, uint32(tp))

}
func (h *Host) Weight() (weight uint32) {
	weight = atomic.LoadUint32(&h.weight)
	return

}
func (h *Host) SetWeight(new uint32) {
	atomic.StoreUint32(&h.weight, new)
}

func (h *Host) Used() (used bool) {
	used = atomic.LoadUint32(&h.used) == 1
	return
}
func (h *Host) SetUsed(new bool) {
	if new {
		atomic.StoreUint32(&h.used, 1)
	} else {
		atomic.StoreUint32(&h.used, 0)
	}
}

func (h *Host) SetDetectorMonitor(d DetectorHostMonitor) {
	h.detector.Store(d)
}

func (h *Host) GetDetectorMonitor() DetectorHostMonitor {
	return h.detector.Load().(DetectorHostMonitor)
}
