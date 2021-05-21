package upstream

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
)

type Result int

type ejectType int

const (
	ejectSuccessRate ejectType = iota
	ejectConsecutiveConnectionError
	ejectConsecutiveError
)

func (t ejectType) String() string {
	switch t {
	case ejectSuccessRate:
		return "Eject Reason:SuccessRate"
	case ejectConsecutiveConnectionError:
		return "Eject Reason:ConsecutiveConnectionError"
	case ejectConsecutiveError:
		return "Eject Reason:ConsecutiveError"
	}
	return "Eject Reason:Unknown"

}

type ChangeStateCallback func(*Host)

type DetectorHostMonitor interface {
	NumEjections() uint32
	PutResult(Result)
	PutResponseTime(time.Duration)
	LastEjectionTime() time.Time
	LastUnejectionTime() time.Time
}

type Detector interface {
	AddChangedStateCallback(ChangeStateCallback)
	SuccessRateAverage() float64
	SuccessRateEjectionThreshold() float64
}

type hostSuccessRatePair struct {
	host        *Host
	successRate float64
}
type successRateAccumelatorBucket struct {
	successRequestCounter uint64
	totalRequestCounter   uint64
}

type successRateAccumulator struct {
	currentSuccessRateBucket atomic.Value
	backupSuccessRateBucket  atomic.Value
}

func newSuccessRateAccumulator() *successRateAccumulator {
	current := successRateAccumelatorBucket{}
	backup := successRateAccumelatorBucket{}
	sm := successRateAccumulator{}
	sm.backupSuccessRateBucket.Store(&backup)
	sm.currentSuccessRateBucket.Store(&current)
	return &sm

}

func (sa *successRateAccumulator) UpdateCurrentWriter() *successRateAccumelatorBucket {
	backup := sa.backupSuccessRateBucket.Load().(*successRateAccumelatorBucket)
	atomic.StoreUint64(&backup.successRequestCounter, 0)
	atomic.StoreUint64(&backup.totalRequestCounter, 0)
	current := sa.currentSuccessRateBucket.Load().(*successRateAccumelatorBucket)
	sa.currentSuccessRateBucket.Store(backup)
	sa.backupSuccessRateBucket.Store(current)
	return sa.currentSuccessRateBucket.Load().(*successRateAccumelatorBucket)
}

func (sa *successRateAccumulator) GetSuccessRate(volume uint64) float64 {
	total := atomic.LoadUint64(&sa.backupSuccessRateBucket.Load().(*successRateAccumelatorBucket).totalRequestCounter)
	if total < volume {
		return -1.0
	}
	success := atomic.LoadUint64(&sa.backupSuccessRateBucket.Load().(*successRateAccumelatorBucket).successRequestCounter)
	return 100.0 * float64(success) / float64(total)
}

type SimpleDetectorHostMonitor struct {
	detector                   *SimpleDetector
	host                       *Host
	consecutiveError           uint64
	consecutiveConnectionError uint64
	lastEjectionTime           atomic.Value
	lastUnejectionTime         atomic.Value
	numEjections               uint32
	successAccumulator         *successRateAccumulator
	// successAccumulatorBucket   *successRateAccumelatorBucket
	successAccumulatorBucket atomic.Value
	// successRate                float64
	successRate atomic.Value
}

func NewSimpleDetectoHostMonitor(detector *SimpleDetector, host *Host) *SimpleDetectorHostMonitor {
	m := &SimpleDetectorHostMonitor{
		detector:                   detector,
		host:                       host,
		consecutiveError:           0,
		consecutiveConnectionError: 0,
		lastUnejectionTime:         atomic.Value{},
		lastEjectionTime:           atomic.Value{},
		numEjections:               0,
		successAccumulator:         newSuccessRateAccumulator(),
	}
	m.lastUnejectionTime.Store(time.Time{})
	m.lastUnejectionTime.Store(time.Time{})
	m.successRate.Store(float64(0.0))
	m.UpdateCurrentSuccessRateBucket()
	return m
}

func (hm *SimpleDetectorHostMonitor) Eject(t time.Time) {
	hm.host.HealthFlagSet(FailedDetectorCheck)
	atomic.AddUint32(&hm.numEjections, 1)
	hm.lastEjectionTime.Store(t)
}
func (hm *SimpleDetectorHostMonitor) Uneject(t time.Time) {
	hm.lastUnejectionTime.Store(t)
}

func (hm *SimpleDetectorHostMonitor) UpdateCurrentSuccessRateBucket() {
	hm.successAccumulatorBucket.Store(hm.successAccumulator.UpdateCurrentWriter())
}

func (hm *SimpleDetectorHostMonitor) NumEjections() uint32 {
	return atomic.LoadUint32(&hm.numEjections)
}
func (hm *SimpleDetectorHostMonitor) PutResponseTime(time.Duration) {
}

func (hm *SimpleDetectorHostMonitor) PutResult(r Result) {
	bucket := hm.successAccumulatorBucket.Load().(*successRateAccumelatorBucket)
	atomic.AddUint64(&bucket.totalRequestCounter, 1)
	//Success
	if r == 0 {
		atomic.AddUint64(&bucket.successRequestCounter, 1)
		atomic.StoreUint64(&hm.consecutiveError, 0)
		atomic.StoreUint64(&hm.consecutiveConnectionError, 0)
		return
	}
	// ConnectionError (Refused, ConnectTimeout)
	if r <= 100 {
		cnt := atomic.AddUint64(&hm.consecutiveConnectionError, 1)
		if cnt == hm.detector.config.ConsecutiveConnectionError {
			hm.detector.OnConsecutiveConnectionFailure(hm.host)
			atomic.StoreUint64(&hm.consecutiveConnectionError, 0)
		}
	} else {
		atomic.StoreUint64(&hm.consecutiveConnectionError, 0)

	}
	// CommonError (Request Timeout)
	if r > 100 && r <= 200 {
		// hm.consecutiveError++
		cnt := atomic.AddUint64(&hm.consecutiveError, 1)
		if cnt == hm.detector.config.ConsecutiveError {
			hm.detector.OnConsecutiveFailure(hm.host)
			atomic.StoreUint64(&hm.consecutiveError, 0)
		}

	} else {
		atomic.StoreUint64(&hm.consecutiveError, 0)
	}
}
func (hm *SimpleDetectorHostMonitor) LastEjectionTime() time.Time {
	return hm.lastEjectionTime.Load().(time.Time)
}
func (hm *SimpleDetectorHostMonitor) LastUnejectionTime() time.Time {
	return hm.lastUnejectionTime.Load().(time.Time)
}

func (hm *SimpleDetectorHostMonitor) SuccessRate() float64 {
	return hm.successRate.Load().(float64)
}

func (hm *SimpleDetectorHostMonitor) SetSuccessRate(newSuccessRate float64) {
	hm.successRate.Store(newSuccessRate)
}

func (hm *SimpleDetectorHostMonitor) SuccessRateAccumulator() *successRateAccumulator {
	return hm.successAccumulator
}
func (hm *SimpleDetectorHostMonitor) ResetConsecutiveError() {
	atomic.StoreUint64(&hm.consecutiveError, 0)
}
func (hm *SimpleDetectorHostMonitor) ResetConsecutiveConnectionError() {
	atomic.StoreUint64(&hm.consecutiveConnectionError, 0)
}

type SimpleDetector struct {
	cluster  *Cluster
	config   config.Detector
	exitChan chan struct{}

	callbacks    []ChangeStateCallback
	callbackMuex *sync.RWMutex

	hostMonitors  map[*Host]*SimpleDetectorHostMonitor
	monitorsMutex *sync.RWMutex

	successRateAverage           atomic.Value
	successRateEjectionThreshold atomic.Value
	hostEjectedNum               int32
}

func NewSimpleDector(c *Cluster, conf config.Detector) *SimpleDetector {
	d := &SimpleDetector{
		cluster:        c,
		config:         conf,
		hostMonitors:   make(map[*Host]*SimpleDetectorHostMonitor),
		monitorsMutex:  new(sync.RWMutex),
		callbacks:      make([]ChangeStateCallback, 0),
		callbackMuex:   new(sync.RWMutex),
		exitChan:       make(chan struct{}),
		hostEjectedNum: 0,
	}
	d.successRateAverage.Store(float64(-1.0))
	d.successRateEjectionThreshold.Store(float64(-1.0))
	d.initialize()
	go d.Start()
	return d
}

func (sd *SimpleDetector) initialize() {
	sd.cluster.hostSet.AddUpdateCallback(func(added, removed []*Host) {
		for _, h := range added {
			sd.addHostMonitor(h)
		}
		for _, h := range removed {
			sd.monitorsMutex.Lock()
			delete(sd.hostMonitors, h)
			sd.monitorsMutex.Unlock()
		}

	})
}

func (sd *SimpleDetector) AddChangedStateCallback(cb ChangeStateCallback) {
	sd.callbackMuex.Lock()
	sd.callbacks = append(sd.callbacks, cb)
	sd.callbackMuex.Unlock()
}

func (sd *SimpleDetector) SuccessRateAverage() float64 {
	return sd.successRateAverage.Load().(float64)

}
func (sd *SimpleDetector) SuccessRateEjectionThreshold() float64 {
	return sd.successRateEjectionThreshold.Load().(float64)

}

func (sd *SimpleDetector) addHostMonitor(h *Host) {
	monitor := NewSimpleDetectoHostMonitor(sd, h)
	sd.monitorsMutex.Lock()
	sd.hostMonitors[h] = monitor
	sd.monitorsMutex.Unlock()
	h.SetDetectorMonitor(monitor)
}

func (sd *SimpleDetector) CheckHostForUneject(h *Host, m *SimpleDetectorHostMonitor, now time.Time) {
	if !h.HealthFlagGet(FailedDetectorCheck) {
		return
	}
	baseEnjectTime := time.Duration(sd.config.BaseEjectionDuration)
	if baseEnjectTime*time.Duration(m.NumEjections()) < now.Sub(m.LastUnejectionTime()) {
		h.HealthFlagClear(FailedDetectorCheck)
		m.ResetConsecutiveError()
		m.ResetConsecutiveConnectionError()
		m.Uneject(now)
		atomic.AddInt32(&sd.hostEjectedNum, -1)
		sd.runCallbacks(h)
		logUneject(h)
	}
}

func (sd *SimpleDetector) Start() {
	ticker := time.NewTicker(time.Duration(sd.config.DetectInterval))
	defer ticker.Stop()
	for {
		select {
		case <-sd.exitChan:
			return
		case <-ticker.C:
			now := time.Now()
			sd.monitorsMutex.RLock()
			type hostMonitorPair struct {
				h *Host
				m *SimpleDetectorHostMonitor
			}
			pairs := make([]hostMonitorPair, 0, len(sd.hostMonitors))
			for h, m := range sd.hostMonitors {
				pairs = append(pairs, hostMonitorPair{
					h: h,
					m: m,
				})

			}
			sd.monitorsMutex.RUnlock()
			for _, p := range pairs {
				sd.CheckHostForUneject(p.h, p.m, now)
				p.m.UpdateCurrentSuccessRateBucket()
				p.m.SetSuccessRate(-1.0)
			}
			sd.ProcessSuccessRateEjections()
		}

	}

}

func (sd *SimpleDetector) EjectHost(h *Host, t ejectType) {
	maxEjectionPercent := sd.config.MaxEjectionPercent
	sd.monitorsMutex.RLock()
	ejectPercent := uint64(100.0 * float64(atomic.LoadInt32(&sd.hostEjectedNum)+1) / float64(len(sd.hostMonitors)))
	sd.monitorsMutex.RUnlock()
	if ejectPercent < maxEjectionPercent {
		sd.monitorsMutex.RLock()
		m, ok := sd.hostMonitors[h]
		sd.monitorsMutex.RUnlock()
		if ok {
			atomic.AddInt32(&sd.hostEjectedNum, 1)
			m.Eject(time.Now())
			sd.runCallbacks(h)
			logEject(h, sd, t)
		}
	}
}
func (sd *SimpleDetector) ProcessSuccessRateEjections() {
	minHostNum := int(sd.config.SuccessRateMinHosts)
	requestVolume := sd.config.SuccessRateRequestVolume
	successRateSum := 0.0
	sd.successRateAverage.Store(float64(-1.0))
	sd.successRateEjectionThreshold.Store(float64(-1.0))
	sd.monitorsMutex.RLock()
	if len(sd.hostMonitors) < minHostNum {
		sd.monitorsMutex.RUnlock()
		return
	}
	validSuccessRateHosts := make([]hostSuccessRatePair, len(sd.hostMonitors))
	for h, m := range sd.hostMonitors {
		if !h.HealthFlagGet(FailedDetectorCheck) {
			hostSuccessRate := m.SuccessRateAccumulator().GetSuccessRate(requestVolume)
			if hostSuccessRate > 0 {
				validSuccessRateHosts = append(validSuccessRateHosts,
					hostSuccessRatePair{
						host:        h,
						successRate: hostSuccessRate,
					})
				successRateSum += hostSuccessRate
				m.SetSuccessRate(hostSuccessRate)
			}

		}

	}
	sd.monitorsMutex.RUnlock()
	if len(validSuccessRateHosts) >= minHostNum {
		rateStdevFactor := sd.config.SuccessRateStdevFactor / 1000
		mean, threshold := successRateEjectionThreshold(successRateSum, validSuccessRateHosts, rateStdevFactor)
		sd.successRateAverage.Store(mean)
		sd.successRateEjectionThreshold.Store(threshold)
		for _, h := range validSuccessRateHosts {
			if h.successRate < threshold {
				sd.EjectHost(h.host, ejectSuccessRate)
			}

		}

	}
}

func (sd *SimpleDetector) runCallbacks(h *Host) {
	sd.callbackMuex.RLock()
	callbacks := make([]ChangeStateCallback, 0, len(sd.callbacks))
	copy(callbacks, sd.callbacks)
	sd.callbackMuex.RUnlock()
	for _, c := range callbacks {
		c(h)
	}

}

func (sd *SimpleDetector) OnConsecutiveConnectionFailure(host *Host) {
	if host.HealthFlagGet(FailedDetectorCheck) {
		return
	}
	sd.EjectHost(host, ejectConsecutiveConnectionError)

}

func (sd *SimpleDetector) OnConsecutiveFailure(host *Host) {
	if host.HealthFlagGet(FailedDetectorCheck) {
		return
	}
	sd.EjectHost(host, ejectConsecutiveError)
}

func successRateEjectionThreshold(successRateSum float64, validHosts []hostSuccessRatePair, stdevFactor float64) (float64, float64) {
	mean := successRateSum / float64(len(validHosts))
	variance := 0.0
	for _, h := range validHosts {
		diff := h.successRate - mean
		variance += diff * diff
	}
	variance = variance / float64(len(validHosts))
	stdev := math.Sqrt(variance)
	return mean, mean - stdevFactor*stdev
}

func logUneject(h *Host) {
	logging.Infow("host uneject", "secs_since_last_action", h.GetDetectorMonitor().LastUnejectionTime().Unix(), "host", h.Address(), "num_ejections", h.GetDetectorMonitor().NumEjections())
}

func logEject(h *Host, detector Detector, tp ejectType) {
	logging.Infow("host eject", "secs_since_last_action", h.GetDetectorMonitor().LastEjectionTime(), "host", h.Address(),
		"num_ejections", h.GetDetectorMonitor().NumEjections(), "eject_reason", tp.String(), "host_success_rate", 0, "cluster_averate_success_rate", detector.SuccessRateAverage(), "cluster_eject_threshold", detector.SuccessRateEjectionThreshold())

}
