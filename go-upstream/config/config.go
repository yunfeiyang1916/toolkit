package config

import (
	"fmt"
	"time"

	"github.com/yunfeiyang1916/toolkit/logging"
)

type Log struct {
}

type Registry struct {
	Backend string
	Static  Static
	File    File
	Consul  Consul
	Timeout time.Duration
	Retry   time.Duration
}

type Static struct {
	Routes string
}

type File struct {
	Path string
}

type Consul struct {
	Addr   string
	Scheme string
	Token  string

	Logger *logging.Logger
}

type Register struct {
	ServiceName string
	ServiceAddr string
	ServicePort int

	ServiceCheckDSN                   string
	ServiceCheckIntervalMs            int
	ServiceCheckTimeoutMs             int
	DeregisterCriticalServiceAfterSec int

	ServiceTags    map[string]string
	DerigesterCh   chan bool
	TagsOverrideCh chan []string
	TagsWatchPath  string
}

func NewRegister(name, addr string, port int) *Register {
	return &Register{
		ServiceName:                       name,
		ServiceAddr:                       addr,
		ServicePort:                       port,
		TagsOverrideCh:                    make(chan []string, 1),
		ServiceCheckDSN:                   fmt.Sprintf("tcp://%s:%d", addr, port),
		ServiceCheckTimeoutMs:             100,
		ServiceCheckIntervalMs:            5000,
		DeregisterCriticalServiceAfterSec: 60,
	}
}

type Cluster struct {
	// 兼容rpc-go ServiceClient数据结构
	Name            string `toml:"service_name"`
	LBType          string `toml:"balancetype"`
	EndpointsFrom   string `toml:"endpoints_from"`
	StaticEndpoints string `toml:"endpoints"`
	Proto           string `toml:"proto"`

	// checker config
	CheckInterval      Duration `toml:"check_interval"`
	UnHealthyThreshold uint32   `toml:"check_unhealth_threshold"`
	HealthyThreshold   uint32   `toml:"check_healthy_threshold"`

	// lb advance config
	LBPanicThreshold int        `toml:"lb_panic_threshold"`
	LBSubsetKeys     [][]string `toml:"lb_subset_selectors"`
	LBDefaultKeys    []string   `toml:"lb_default_keys"`

	// detector config
	Detector Detector `toml:"detector"`

	Datacenter string `toml:"datacenter"`
}

type Detector struct {
	DetectInterval             Duration `toml:"detect_interval"`
	BaseEjectionDuration       Duration `toml:"base_ejection_duration"`
	ConsecutiveError           uint64   `toml:"consecutive_error"`
	ConsecutiveConnectionError uint64   `toml:"consecutive_connect_error"`
	MaxEjectionPercent         uint64   `toml:"max_ejection_percent"`
	SuccessRateMinHosts        uint64   `toml:"success_rate_min_hosts"`
	SuccessRateRequestVolume   uint64   `toml:"success_rate_request_volume"`
	SuccessRateStdevFactor     float64  `toml:"success_rate_stdev_factor"`
	EnforcingSuccessRate       uint64   `toml:"enforcing_success_rate"`
}

func NewCluster() Cluster {
	return Cluster{
		LBType:             "WeightRoundRobin",
		CheckInterval:      Duration(5 * time.Second),
		UnHealthyThreshold: 10,
		HealthyThreshold:   3,
		LBPanicThreshold:   0,
		Detector: Detector{
			DetectInterval:             Duration(20 * time.Second),
			BaseEjectionDuration:       Duration(1 * time.Second),
			ConsecutiveError:           20,
			ConsecutiveConnectionError: 5,
			MaxEjectionPercent:         50,
			SuccessRateMinHosts:        3,
			SuccessRateRequestVolume:   100,
			SuccessRateStdevFactor:     1900,
		},
	}
}

type Duration time.Duration

// struct {
// }

func (d *Duration) UnmarshalText(text []byte) error {
	duration, err := time.ParseDuration(string(text))
	*d = Duration(duration)
	return err
}
