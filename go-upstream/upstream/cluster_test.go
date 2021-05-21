package upstream

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry/consul"
)

func TestCluster(t *testing.T) {
	var cfg = &config.Consul{
		Addr:   "ali-a-inf-consul03.bj:8500",
		Scheme: "http",
		Token:  "",
		// ServiceName:                       "test",
		// ServiceAddr:                       "",
		// ServicePort:                       8300,
		// ServiceTags:                       []string{"stage=previed", "type=std"},
		// ServiceCheckDSN:                   "tcp://127.0.0.1:8500",
		// ServiceCheckIntervalMs:            1000,
		// ServiceCheckTimeoutMs:             200,
		// DeregisterCriticalServiceAfterSec: 60,
	}
	be, _ := consul.NewBackend(cfg)
	cc := config.Cluster{
		//Name:               "upload.front.resource-http",
		Name:               "media.dispatch.LiveStream-http",
		CheckInterval:      config.Duration(1 * time.Second),
		UnHealthyThreshold: 3,
		HealthyThreshold:   2,
		LBPanicThreshold:   4,
		LBType:             "WeightRoundRobin",
		//LBSubsetKeys:       [][]string{[]string{"a"}},
		LBDefaultKeys: []string{},
		Detector: config.Detector{
			DetectInterval:             config.Duration(20 * time.Second),
			BaseEjectionDuration:       config.Duration(100 * time.Millisecond),
			ConsecutiveError:           5,
			ConsecutiveConnectionError: 2,
			MaxEjectionPercent:         50,
			SuccessRateMinHosts:        1,
			SuccessRateRequestVolume:   100,
			SuccessRateStdevFactor:     1900,
		},
	}
	cluster := NewCluster(cc, be)
	_ = cluster
	//for i := 0; i < 1; i++ {

	//	go func() {
	//		for {
	//			ctx := InjectSubsetCarrier(context.Background(), []string{"a", "b"})
	//			host := cluster.Balancer().ChooseHost(ctx)
	//			if host != nil {
	//				//fmt.Printf("get host %s\n", host.Address())
	//				logging.Infof("get host %s", host.Address())
	//				host.GetDetectorMonitor().PutResult(Result(103))
	//			}
	//			time.Sleep(1 * time.Millisecond)
	//		}
	//	}()
	//}
	time.Sleep(100 * time.Second)

}

func BenchmarkCluster(b *testing.B) {
	var cfg = &config.Consul{
		Addr:   "ali-a-inf-consul03.bj:8500",
		Scheme: "http",
		Token:  "",
		// ServiceName:                       "test",
		// ServiceAddr:                       "",
		// ServicePort:                       8300,
		// ServiceTags:                       []string{"stage=previed", "type=std"},
		// ServiceCheckDSN:                   "tcp://127.0.0.1:8500",
		// ServiceCheckIntervalMs:            1000,
		// ServiceCheckTimeoutMs:             200,
		// DeregisterCriticalServiceAfterSec: 60,
	}
	be, _ := consul.NewBackend(cfg)
	cc := config.Cluster{
		Name:               "upload.front.resource-http",
		CheckInterval:      config.Duration(1 * time.Second),
		UnHealthyThreshold: 3,
		HealthyThreshold:   2,
		LBPanicThreshold:   4,
		LBType:             "WeightRoundRobin",
		LBSubsetKeys:       [][]string{[]string{"a"}},
		LBDefaultKeys:      []string{"a", "b"},
		Detector: config.Detector{
			DetectInterval:             config.Duration(10 * time.Second),
			BaseEjectionDuration:       config.Duration(20 * time.Second),
			ConsecutiveError:           5,
			ConsecutiveConnectionError: 2,
			MaxEjectionPercent:         50,
			SuccessRateMinHosts:        1,
			SuccessRateRequestVolume:   100,
			SuccessRateStdevFactor:     1900,
		},
	}
	cluster := NewCluster(cc, be)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := InjectSubsetCarrier(context.Background(), []string{"a", "b"})
		host := cluster.Balancer().ChooseHost(ctx)
		_ = host
		host.GetDetectorMonitor().PutResult(Result(rand.Intn(300)))
	}
}

func TestNewCluster(t *testing.T) {
	type args struct {
		conf    config.Cluster
		backend registry.Backend
	}
	tests := []struct {
		name string
		args args
		want *Cluster
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCluster(tt.args.conf, tt.args.backend); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCluster() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_HealthChecker(t *testing.T) {
	type fields struct {
		name            string
		checker         *HealthChecker
		registerBackend registry.Backend
		hostSet         *HostSet
		maxWeight       uint32
	}
	tests := []struct {
		name   string
		fields fields
		want   *HealthChecker
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				name:            tt.fields.name,
				checker:         tt.fields.checker,
				registerBackend: tt.fields.registerBackend,
				hostSet:         tt.fields.hostSet,
				maxWeight:       tt.fields.maxWeight,
			}
			if got := c.HealthChecker(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Cluster.HealthChecker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_listenConfigChange(t *testing.T) {
	type fields struct {
		name            string
		checker         *HealthChecker
		registerBackend registry.Backend
		hostSet         *HostSet
		maxWeight       uint32
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				name:            tt.fields.name,
				checker:         tt.fields.checker,
				registerBackend: tt.fields.registerBackend,
				hostSet:         tt.fields.hostSet,
				maxWeight:       tt.fields.maxWeight,
			}
			c.listenConfigChange()
		})
	}
}

func TestCluster_onClusterChanged(t *testing.T) {
	type fields struct {
		name            string
		checker         *HealthChecker
		registerBackend registry.Backend
		hostSet         *HostSet
		maxWeight       uint32
	}
	type args struct {
		newCluster *registry.Cluster
	}

	var cfg = &config.Consul{
		Addr:   "127.0.0.1:8500",
		Scheme: "http",
		Token:  "",
	}

	hs1 := []*Host{NewHost("1.1.1.1:1234", 100, nil)}
	hs2 := []*Host{NewHost("2.2.2.2:1234", 100, nil)}
	hs3 := []*Host{NewHost("3.3.3.3:1234", 100, nil)}

	end1 := registry.Endpoint{"1", "1.1.1.1", 1234, nil}
	cluster1 := &registry.Cluster{
		Name:      "aaaa",
		Endpoints: []registry.Endpoint{end1},
	}

	end2 := registry.Endpoint{"2", "2.2.2.2", 1234, nil}
	cluster2 := &registry.Cluster{
		Name:      "bbbb",
		Endpoints: []registry.Endpoint{end2},
	}

	end3 := registry.Endpoint{"3", "3.3.3.3", 1234, nil}
	cluster3 := &registry.Cluster{
		Name:      "cccc",
		Endpoints: []registry.Endpoint{end3},
	}

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
		{name: "aaaa", fields: fields{name: "aaaa", hostSet: NewHostSet(hs1, hs1)}, args: args{newCluster: cluster1}},
		{name: "bbbb", fields: fields{name: "bbbb", hostSet: NewHostSet(hs2, hs2)}, args: args{newCluster: cluster2}},
		{name: "cccc", fields: fields{name: "cccc", hostSet: NewHostSet(hs3, hs3)}, args: args{newCluster: cluster3}},
	}
	be, _ := consul.NewBackend(cfg)
	c := NewCluster(config.NewCluster(), be)
	c.AddHostChangedCallback(func(current []string, added []string, removed []string) {
		fmt.Println("-------------------")
		fmt.Println("current:", current)
		fmt.Println("added:", added)
		fmt.Println("removed:", removed)
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//c := &Cluster{
			//	name:            tt.fields.name,
			//	checker:         tt.fields.checker,
			//	registerBackend: tt.fields.registerBackend,
			//	hostSet:         tt.fields.hostSet,
			//	maxWeight:       tt.fields.maxWeight,
			//}
			c.onClusterChanged(tt.args.newCluster)
			time.Sleep(100 * time.Millisecond)
		})
	}
	time.Sleep(1 * time.Second)
}

func TestCluster_updateSet(t *testing.T) {
	type fields struct {
		name            string
		checker         *HealthChecker
		registerBackend registry.Backend
		hostSet         *HostSet
		maxWeight       uint32
	}
	type args struct {
		current []*Host
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
			c := &Cluster{
				name:            tt.fields.name,
				checker:         tt.fields.checker,
				registerBackend: tt.fields.registerBackend,
				hostSet:         tt.fields.hostSet,
				maxWeight:       tt.fields.maxWeight,
			}
			c.updateSet(tt.args.current, tt.args.added, tt.args.removed)
		})
	}
}

func TestCluster_updateDynamicHostList(t *testing.T) {
	type fields struct {
		name            string
		checker         *HealthChecker
		registerBackend registry.Backend
		hostSet         *HostSet
		maxWeight       uint32
	}
	type args struct {
		newHost      []*Host
		added        *[]*Host
		removed      *[]*Host
		current      *[]*Host
		updatedHosts map[string]*Host
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				name:            tt.fields.name,
				checker:         tt.fields.checker,
				registerBackend: tt.fields.registerBackend,
				hostSet:         tt.fields.hostSet,
				maxWeight:       tt.fields.maxWeight,
			}
			if got := c.updateDynamicHostList(tt.args.newHost, tt.args.added, tt.args.removed, tt.args.current, tt.args.updatedHosts); got != tt.want {
				t.Errorf("Cluster.updateDynamicHostList() = %v, want %v", got, tt.want)
			}
		})
	}
}
