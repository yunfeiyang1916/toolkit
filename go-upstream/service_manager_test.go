package upstream

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry/consul"
	"github.com/yunfeiyang1916/toolkit/go-upstream/upstream"
	"github.com/yunfeiyang1916/toolkit/logging"
)

func TestServiceManager(t *testing.T) {
	cfg := config.Consul{
		Addr:   "127.0.0.1:8500",
		Scheme: "http",
		Token:  "",
		Logger: logging.New(),
	}
	var err error
	registry.Default, err = consul.NewBackend(&cfg)
	if err != nil {
		t.Fatalf("init consul backend error %s", err)
	}
	reg := config.NewRegister("test", "127.0.0.1", 8500)
	reg.TagsWatchPath = "/test/tags"
	sm := registry.NewServiceManager(cfg.Logger)
	sm.Register(reg)
	service := "app.service.name.file-http"
	// time.Sleep(120 * time.Second)
	cc := config.Cluster{
		Name:          service,
		EndpointsFrom: "consul",

		CheckInterval:      config.Duration(1 * time.Second),
		UnHealthyThreshold: 3,
		HealthyThreshold:   2,
		LBPanicThreshold:   4,
		LBType:             "WeightRoundRobin",
		LBSubsetKeys:       [][]string{[]string{"a"}},
		LBDefaultKeys:      []string{"a", "1234"},
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
	manager := upstream.NewClusterManager()
	manager.InitService(cc)
	for i := 0; i < 10; i++ {

		go func() {
			for {
				ctx := upstream.InjectSubsetCarrier(context.Background(), []string{})
				host := manager.ChooseHost(ctx, service)
				if host != nil {
					fmt.Printf("get host %s\n", host.Address())
				}
				// host.GetDetectorMonitor().PutResult(Result(rand.Intn(300)))
				manager.PutResult(service, host.Address(), rand.Intn(300))
			}
		}()
	}
	time.Sleep(120 * time.Second)
	sm.Deregister()
}
