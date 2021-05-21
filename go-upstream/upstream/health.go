package upstream

import (
	"net"
	"strings"
	"sync"
	"time"
)

type HealthTransition int
type HealthFlag uint32
type ActiveHalthFailureType uint32
type HealthCheckFailureType int
type HealthCheckerType int

const (
	Unchanged HealthTransition = iota
	Changed
	ChangePending
)

const (
	ACTIVE HealthCheckFailureType = iota
	PASSIVE
	NETWORK
)

const (
	HTTP HealthCheckerType = iota
	TCP
)

const (
	FailedActiveHC       HealthFlag = 0x01
	FailedDetectorCheck  HealthFlag = 0x02
	FailedRegistryHealth HealthFlag = 0x04
)

const (
	Unknown ActiveHalthFailureType = iota
	UnHealthy
	Timeout
)

type HealthChecker struct {
	name               string
	checkInterval      time.Duration
	unHealthyThreshold uint32
	healthyThreshold   uint32
	mu                 sync.Mutex
	activeSessions     map[*Host]*ActiveHealthCheckSession
	completeCallbacks  []HealthCheckCompleteCallback
}

func (t ActiveHalthFailureType) String() string {
	switch t {
	case UnHealthy:
		return "UnHealthy"
	case Timeout:
		return "Timeout"
	}
	return "Unknown"
}

func (t HealthTransition) String() string {
	switch t {
	case Changed:
		return "Changed"
	case ChangePending:
		return "ChangePending"
	}
	return "Unchanged"

}

type HealthCheckCompleteCallback func(*Host, HealthTransition)

func NewHealthChecker(tp HealthCheckerType, interval time.Duration, unHealthyThreshold, healthyThreshold uint32, name string) *HealthChecker {
	return &HealthChecker{
		name:               name,
		checkInterval:      interval,
		unHealthyThreshold: unHealthyThreshold,
		healthyThreshold:   healthyThreshold,
		activeSessions:     make(map[*Host]*ActiveHealthCheckSession),
		completeCallbacks:  make([]HealthCheckCompleteCallback, 0),
	}
}

func (hc *HealthChecker) AddHosts(added []*Host) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	for _, h := range added {
		h.SetActiveHealthFailureType(Unknown)
		session := newActiveHealthCheckSession(h, hc)
		hc.activeSessions[h] = session
		session.Start()
	}
}

func (hc *HealthChecker) OnHostsChanged(added []*Host, removed []*Host) {
	hc.AddHosts(added)
	hc.mu.Lock()
	defer hc.mu.Unlock()
	for _, h := range removed {
		if session, ok := hc.activeSessions[h]; ok {
			delete(hc.activeSessions, h)
			session.Close()
		}
	}
}

func (hc *HealthChecker) AddHostCheckCompleteCb(cb HealthCheckCompleteCallback) {
	hc.mu.Lock()
	hc.completeCallbacks = append(hc.completeCallbacks, cb)
	hc.mu.Unlock()
}

func (hc *HealthChecker) onStateChange(h *Host, state HealthTransition) {
	// Notify hostSet
	logging.Infof("%s checker %p, host %s state changed %s, state %s, health? %t", hc.name, hc, h.Address(), state.String(), h.GetActiveHealthFailureType().String(), h.Healthy())
	callbacks := make([]HealthCheckCompleteCallback, 0)
	hc.mu.Lock()
	callbacks = append(callbacks, hc.completeCallbacks...)
	hc.mu.Unlock()
	for _, cb := range callbacks {
		cb(h, state)
	}
}

func (hc *HealthChecker) check(h *Host, session *ActiveHealthCheckSession) {
	address := h.Address()
	conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
	if err != nil {
		errUpper := strings.ToUpper(err.Error())
		if strings.Contains(errUpper, "TIMEOUT") {
			h.SetActiveHealthFailureType(Timeout)
		} else {
			h.SetActiveHealthFailureType(UnHealthy)
		}
		session.handleFailure(NETWORK)
		return
	}
	conn.Close()
	session.handleSuccess()
	h.SetActiveHealthFailureType(Unknown)
}

type ActiveHealthCheckSession struct {
	host         *Host
	timer        *time.Timer
	checker      *HealthChecker
	numUnHealthy uint32
	numHealthy   uint32
	firstCheck   bool
	exit         chan struct{}
	mu           sync.Mutex
}

func newActiveHealthCheckSession(host *Host, checker *HealthChecker) *ActiveHealthCheckSession {
	return &ActiveHealthCheckSession{
		host:         host,
		timer:        nil,
		checker:      checker,
		firstCheck:   true,
		exit:         make(chan struct{}),
		numHealthy:   0,
		numUnHealthy: 0,
	}
}

func (ahcs *ActiveHealthCheckSession) Start() {
	go func() {
		ahcs.checker.check(ahcs.host, ahcs)
		ticker := time.NewTicker(ahcs.checker.checkInterval)
		defer ticker.Stop()
		// fire ticker imediatilly
		ahcs.checker.check(ahcs.host, ahcs)
		for {
			select {
			case <-ahcs.exit:
				return
			case <-ticker.C:
				ahcs.checker.check(ahcs.host, ahcs)
			}
		}
	}()
}

func (ahcs *ActiveHealthCheckSession) Close() {
	close(ahcs.exit)
}

func (ahcs *ActiveHealthCheckSession) SetUnhealthy(tp HealthCheckFailureType) HealthTransition {
	ahcs.mu.Lock()
	defer ahcs.mu.Unlock()
	return ahcs.setUnhealthyUnSafe(tp)
}

func (ahcs *ActiveHealthCheckSession) setUnhealthyUnSafe(tp HealthCheckFailureType) HealthTransition {
	ahcs.numHealthy = 0
	changeState := Unchanged
	if !ahcs.host.HealthFlagGet(FailedActiveHC) {
		if tp != NETWORK || incr(&ahcs.numUnHealthy) == ahcs.checker.unHealthyThreshold {
			ahcs.host.HealthFlagSet(FailedActiveHC)
			changeState = Changed
		} else {
			changeState = ChangePending
		}

	}
	ahcs.firstCheck = false
	ahcs.checker.onStateChange(ahcs.host, changeState)
	return changeState
}

func (ahcs *ActiveHealthCheckSession) handleSuccess() {
	ahcs.mu.Lock()
	ahcs.numUnHealthy = 0
	changeState := Unchanged
	if ahcs.host.HealthFlagGet(FailedActiveHC) {
		if ahcs.firstCheck || incr(&ahcs.numHealthy) == ahcs.checker.healthyThreshold {
			ahcs.host.HealthFlagClear(FailedActiveHC)
			changeState = Changed
		} else {
			changeState = ChangePending
		}
	}
	ahcs.firstCheck = false
	ahcs.mu.Unlock()
	ahcs.checker.onStateChange(ahcs.host, changeState)
}

func (ahcs *ActiveHealthCheckSession) handleFailure(tp HealthCheckFailureType) {
	ahcs.mu.Lock()
	ahcs.setUnhealthyUnSafe(tp)
	ahcs.mu.Unlock()
}
