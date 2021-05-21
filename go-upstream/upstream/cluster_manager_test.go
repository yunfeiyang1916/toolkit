package upstream

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry/consul"
	"golang.org/x/net/context"
)

func TestNewClusterManager(t *testing.T) {
	var cfg = &config.Consul{
		Addr:   "10.55.4.34:8500",
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
		Name:          "upload.front.resource-http",
		EndpointsFrom: "consul",

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
	registry.Default = be

	manager := NewClusterManager()
	// _ = manager.InitService(cc)
	cc2 := cc
	cc2.LBDefaultKeys = nil
	cc2.EndpointsFrom = "consul"
	cc2.StaticEndpoints = ""
	cc2.Datacenter = "ali-test"
	cc2.Name = "inf.ae.api_service-http"
	err := manager.InitService(cc2)
	if err != nil {
		t.Fatal(err)

	}
	for i := 0; i < 10; i++ {

		go func() {
			for {
				ctx := InjectSubsetCarrier(context.Background(), []string{"a", "b"})
				host := manager.ChooseHost(ctx, cc2.Name)
				if host != nil {
					fmt.Printf("get host %s\n", host.Address())
					manager.PutResult(cc2.Name, host.Address(), rand.Intn(300))
				}
				time.Sleep(100 * time.Second)
				// host.GetDetectorMonitor().PutResult(Result(rand.Intn(300)))
			}
		}()
	}
	time.Sleep(1000 * time.Second)
}

func TestNewClusterManagerFile(t *testing.T) {
	manager := NewClusterManager()
	cc2 := config.Cluster{
		Name:          "upload.front.resource-http",
		EndpointsFrom: "consul",

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
	cc2.LBDefaultKeys = nil
	cc2.EndpointsFrom = "file://discovery/inf.ae.api_service"
	cc2.StaticEndpoints = "127.0.0.1:1000"
	cc2.Datacenter = "ali-test"
	cc2.Name = "inf.ae.api_service-http"
	err := manager.InitService(cc2)
	if err != nil {
		t.Fatal(err)

	}
	for i := 0; i < 10; i++ {

		go func() {
			for {
				ctx := InjectSubsetCarrier(context.Background(), []string{"a", "b"})
				host := manager.ChooseHost(ctx, cc2.Name)
				if host != nil {
					fmt.Printf("get host %s\n", host.Address())
					manager.PutResult(cc2.Name, host.Address(), rand.Intn(300))
				}
				time.Sleep(100 * time.Second)
				// host.GetDetectorMonitor().PutResult(Result(rand.Intn(300)))
			}
		}()
	}
	time.Sleep(1000 * time.Second)
}

func TestClusterManager_ChooseHost(t *testing.T) {
	var cfg = &config.Consul{
		Addr:   "10.55.4.34:8500",
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
		Name:          "nvwa.fps.switch.base-http",
		EndpointsFrom: "consul",
		Proto:         "http",
		//CheckInterval:      config.Duration(1 * time.Second),
		//UnHealthyThreshold: 3,
		//HealthyThreshold:   2,
		//LBPanicThreshold:   4,
		LBType:        "roundrobin",
		LBSubsetKeys:  [][]string{[]string{"env"}, []string{"env", "_namespace_appkey_"}},
		LBDefaultKeys: []string{"env", "online"},
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
	registry.Default = be
	manager := NewClusterManager()
	err := manager.InitService(cc)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("##init##")
	c1 := newCluster("env=online", 0)
	manager.Cluster(cc.Name).onClusterChanged(c1)

	ctx := InjectSubsetCarrier(context.Background(), []string{"env", "online"})
	host := manager.ChooseHost(ctx, cc.Name)
	fmt.Println(host.Address())

	ctx = InjectSubsetCarrier(context.Background(), []string{"env", "preview"})
	host = manager.ChooseHost(ctx, cc.Name)
	fmt.Println(host.Address())

	fmt.Println("##update 2##")
	c3 := newCluster("env=online", 1)
	manager.Cluster(cc.Name).onClusterChanged(c3)

	fmt.Println("##update 1##")
	c5 := newCluster("env=preview", 1)
	manager.Cluster(cc.Name).onClusterChanged(c5)

	fmt.Println("##update 3##")
	c4 := newCluster("env=online", 0)
	manager.Cluster(cc.Name).onClusterChanged(c4)

	for index := 0; index < 4; index++ {
		ctx := InjectSubsetCarrier(context.Background(), []string{"env", "online"})
		host := manager.ChooseHost(ctx, cc.Name)
		fmt.Println("##update 3#online#", index, host.Address())
		ctx = InjectSubsetCarrier(context.Background(), []string{"env", "preview"})
		host = manager.ChooseHost(ctx, cc.Name)
		fmt.Println("##update 3#preview#", index, host.Address())
	}
	//time.Sleep(3 * time.Second)

}

func newCluster(tag string, isAdd int) *registry.Cluster {
	newCluster := &registry.Cluster{
		Name: "nvwa.fps.switch.base-http",
		Endpoints: []registry.Endpoint{
			registry.Endpoint{
				ID:   "nvwa.fps.switch.base-http-10.111.165.25:9996",
				Addr: "10.111.165.25",
				Port: 9996,
				Tags: []string{"__weight=100", "dc=ali-vpc", "env=online"},
			},
			registry.Endpoint{
				ID:   "nvwa.fps.switch.base-http-10.111.174.251:9996",
				Addr: "10.111.174.251",
				Port: 9996,
				Tags: []string{"__weight=100", "dc=ali-vpc", "env=online"},
			},
		},
	}
	if isAdd == 1 {
		newCluster.Endpoints = append(newCluster.Endpoints,
			registry.Endpoint{
				ID:   "nvwa.fps.switch.base-http-10.111.244.237:9996",
				Addr: "10.111.244.237",
				Port: 9996,
				Tags: []string{"__weight=100", "dc=ali-vpc", tag},
			})
	}
	newCluster.AddEnvTag()
	return newCluster
}
