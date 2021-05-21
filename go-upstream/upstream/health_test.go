package upstream

import (
	"net/http"
	_ "net/http/pprof"
	"reflect"
	"testing"
	"time"
)

func TestNewHealthChecker(t *testing.T) {
	type args struct {
		tp                 HealthCheckerType
		interval           time.Duration
		unHealthyThreshold uint32
		healthyThreshold   uint32
	}
	tests := []struct {
		name string
		args args
		want *HealthChecker
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHealthChecker(tt.args.tp, tt.args.interval, tt.args.unHealthyThreshold, tt.args.healthyThreshold, "test"); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHealhtChecker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHealthChecker_AddHosts(t *testing.T) {
	type fields struct {
		checkInterval      time.Duration
		unHealthyThreshold uint32
		healthyThreshold   uint32
		activeSessions     map[*Host]*ActiveHealthCheckSession
	}
	type args struct {
		added []*Host
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := &HealthChecker{
				checkInterval:      tt.fields.checkInterval,
				unHealthyThreshold: tt.fields.unHealthyThreshold,
				healthyThreshold:   tt.fields.healthyThreshold,
				activeSessions:     tt.fields.activeSessions,
			}
			hc.AddHosts(tt.args.added)
		})
	}
}

func TestHealthChecker_HostsChanged(t *testing.T) {
	type fields struct {
		checkInterval      time.Duration
		unHealthyThreshold uint32
		healthyThreshold   uint32
		activeSessions     map[*Host]*ActiveHealthCheckSession
	}
	type args struct {
		added   []*Host
		removed []*Host
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := &HealthChecker{
				checkInterval:      tt.fields.checkInterval,
				unHealthyThreshold: tt.fields.unHealthyThreshold,
				healthyThreshold:   tt.fields.healthyThreshold,
				activeSessions:     tt.fields.activeSessions,
			}
			hc.OnHostsChanged(tt.args.added, tt.args.removed)
		})
	}
}

func TestHealthChecker_stateChange(t *testing.T) {
	type fields struct {
		checkInterval      time.Duration
		unHealthyThreshold uint32
		healthyThreshold   uint32
		activeSessions     map[*Host]*ActiveHealthCheckSession
	}
	type args struct {
		h     *Host
		state HealthTransition
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := &HealthChecker{
				checkInterval:      tt.fields.checkInterval,
				unHealthyThreshold: tt.fields.unHealthyThreshold,
				healthyThreshold:   tt.fields.healthyThreshold,
				activeSessions:     tt.fields.activeSessions,
			}
			hc.onStateChange(tt.args.h, tt.args.state)
		})
	}
}

func TestHealthChecker_check(t *testing.T) {
	type fields struct {
		checkInterval      time.Duration
		unHealthyThreshold uint32
		healthyThreshold   uint32
		activeSessions     map[*Host]*ActiveHealthCheckSession
	}
	type args struct {
		h       *Host
		session *ActiveHealthCheckSession
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := &HealthChecker{
				checkInterval:      tt.fields.checkInterval,
				unHealthyThreshold: tt.fields.unHealthyThreshold,
				healthyThreshold:   tt.fields.healthyThreshold,
				activeSessions:     tt.fields.activeSessions,
			}
			hc.check(tt.args.h, tt.args.session)
		})
	}
}

func Test_newActiveHealthCheckSession(t *testing.T) {
	type args struct {
		host *Host
	}
	tests := []struct {
		name string
		args args
		want *ActiveHealthCheckSession
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newActiveHealthCheckSession(tt.args.host, nil); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newActiveHealthCheckSession() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActiveHealthCheckSession_Start(t *testing.T) {
	type fields struct {
		host         *Host
		timer        *time.Timer
		checker      *HealthChecker
		numUnHealthy uint32
		numHealthy   uint32
		firstCheck   bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ahcs := &ActiveHealthCheckSession{
				host:         tt.fields.host,
				timer:        tt.fields.timer,
				checker:      tt.fields.checker,
				numUnHealthy: tt.fields.numUnHealthy,
				numHealthy:   tt.fields.numHealthy,
				firstCheck:   tt.fields.firstCheck,
			}
			ahcs.Start()
		})
	}
}

func TestActiveHealthCheckSession_Close(t *testing.T) {
	type fields struct {
		host         *Host
		timer        *time.Timer
		checker      *HealthChecker
		numUnHealthy uint32
		numHealthy   uint32
		firstCheck   bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ahcs := &ActiveHealthCheckSession{
				host:         tt.fields.host,
				timer:        tt.fields.timer,
				checker:      tt.fields.checker,
				numUnHealthy: tt.fields.numUnHealthy,
				numHealthy:   tt.fields.numHealthy,
				firstCheck:   tt.fields.firstCheck,
			}
			ahcs.Close()
		})
	}
}

func TestActiveHealthCheckSession_SetUnhealthy(t *testing.T) {
	type fields struct {
		host         *Host
		timer        *time.Timer
		checker      *HealthChecker
		numUnHealthy uint32
		numHealthy   uint32
		firstCheck   bool
	}
	type args struct {
		tp HealthCheckFailureType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   HealthTransition
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ahcs := &ActiveHealthCheckSession{
				host:         tt.fields.host,
				timer:        tt.fields.timer,
				checker:      tt.fields.checker,
				numUnHealthy: tt.fields.numUnHealthy,
				numHealthy:   tt.fields.numHealthy,
				firstCheck:   tt.fields.firstCheck,
			}
			if got := ahcs.SetUnhealthy(tt.args.tp); got != tt.want {
				t.Errorf("ActiveHealthCheckSession.SetUnhealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActiveHealthCheckSession_setUnhealthyUnSafe(t *testing.T) {
	type fields struct {
		host         *Host
		timer        *time.Timer
		checker      *HealthChecker
		numUnHealthy uint32
		numHealthy   uint32
		firstCheck   bool
	}
	type args struct {
		tp HealthCheckFailureType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   HealthTransition
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ahcs := &ActiveHealthCheckSession{
				host:         tt.fields.host,
				timer:        tt.fields.timer,
				checker:      tt.fields.checker,
				numUnHealthy: tt.fields.numUnHealthy,
				numHealthy:   tt.fields.numHealthy,
				firstCheck:   tt.fields.firstCheck,
			}
			if got := ahcs.setUnhealthyUnSafe(tt.args.tp); got != tt.want {
				t.Errorf("ActiveHealthCheckSession.setUnhealthyUnSafe() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActiveHealthCheckSession_handleSuccess(t *testing.T) {
	type fields struct {
		host         *Host
		timer        *time.Timer
		checker      *HealthChecker
		numUnHealthy uint32
		numHealthy   uint32
		firstCheck   bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ahcs := &ActiveHealthCheckSession{
				host:         tt.fields.host,
				timer:        tt.fields.timer,
				checker:      tt.fields.checker,
				numUnHealthy: tt.fields.numUnHealthy,
				numHealthy:   tt.fields.numHealthy,
				firstCheck:   tt.fields.firstCheck,
			}
			ahcs.handleSuccess()
		})
	}
}

func TestActiveHealthCheckSession_handleFailure(t *testing.T) {
	type fields struct {
		host         *Host
		timer        *time.Timer
		checker      *HealthChecker
		numUnHealthy uint32
		numHealthy   uint32
		firstCheck   bool
	}
	type args struct {
		tp HealthCheckFailureType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ahcs := &ActiveHealthCheckSession{
				host:         tt.fields.host,
				timer:        tt.fields.timer,
				checker:      tt.fields.checker,
				numUnHealthy: tt.fields.numUnHealthy,
				numHealthy:   tt.fields.numHealthy,
				firstCheck:   tt.fields.firstCheck,
			}
			ahcs.handleFailure(tt.args.tp)
		})
	}
}

func TestActiveHalthFailureType_String(t *testing.T) {
	tests := []struct {
		name string
		t    ActiveHalthFailureType
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.String(); got != tt.want {
				t.Errorf("ActiveHalthFailureType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHealthChecker_OnHostsChanged(t *testing.T) {
	type fields struct {
		checkInterval      time.Duration
		unHealthyThreshold uint32
		healthyThreshold   uint32
		activeSessions     map[*Host]*ActiveHealthCheckSession
	}
	type args struct {
		added   []*Host
		removed []*Host
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := &HealthChecker{
				checkInterval:      tt.fields.checkInterval,
				unHealthyThreshold: tt.fields.unHealthyThreshold,
				healthyThreshold:   tt.fields.healthyThreshold,
				activeSessions:     tt.fields.activeSessions,
			}
			hc.OnHostsChanged(tt.args.added, tt.args.removed)
		})
	}
}

func TestHealthChecker_onStateChange(t *testing.T) {
	type fields struct {
		checkInterval      time.Duration
		unHealthyThreshold uint32
		healthyThreshold   uint32
		activeSessions     map[*Host]*ActiveHealthCheckSession
	}
	type args struct {
		h     *Host
		state HealthTransition
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := &HealthChecker{
				checkInterval:      tt.fields.checkInterval,
				unHealthyThreshold: tt.fields.unHealthyThreshold,
				healthyThreshold:   tt.fields.healthyThreshold,
				activeSessions:     tt.fields.activeSessions,
			}
			hc.onStateChange(tt.args.h, tt.args.state)
		})
	}
}

func TestHealthChecker(t *testing.T) {
	go http.ListenAndServe("127.0.0.1:22345", nil)
	checher := NewHealthChecker(TCP, 1*time.Second, 5, 4, "test")
	h := NewHost("127.0.0.1:12345", 100, nil)
	checher.AddHosts([]*Host{h})
	time.Sleep(100 * time.Second)
	checher.OnHostsChanged(nil, []*Host{h})
}
