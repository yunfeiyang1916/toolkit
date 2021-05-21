package framework

import (
	"time"

	"github.com/yunfeiyang1916/toolkit/framework/ratelimit"

	"github.com/yunfeiyang1916/toolkit/framework/breaker"

	upstreamconfig "github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/kafka"
	"github.com/yunfeiyang1916/toolkit/sql"
)

type duration struct {
	time.Duration
}

// UnmarshalText 解码text
func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

/*
limit: 限流阈值
open: 限流开关
minsamples: 最小样本数量(采样时间500ms)
error_percent_threshold: 错误比阈值
consecutive_error_threshold: 连续错误数阈值
break: 熔断开关, 当break=true, 当前client调用一直处于熔断状态, 不能对下游服务发起请求
*/

type Resource struct {
	ratelimit.LimiterConfig
	breaker.BreakerConfig
}

type DefaultCircuit struct {
	Server Resource `toml:"server"`
	Client Resource `toml:"client"`
}

// ServerClient 远程服务客户端配置
type ServerClient struct {
	// 应用名称
	APPName *string `toml:"app_name"`
	// 服务发现名称
	ServiceName string `toml:"service_name"`
	Ipport      string `toml:"endpoints"`
	Balancetype string `toml:"balancetype"`
	// 请求协议,比如http
	ProtoType           string `toml:"proto"`
	ConnectTimeout      int    `toml:"connect_timeout"`
	Namespace           string `toml:"namespace"`
	ReadTimeout         int    `toml:"read_timeout"`
	WriteTimeout        int    `toml:"write_timeout"`
	KeepaliveTimeout    int    `toml:"keepalive_timeout"`
	MaxIdleConns        int    `toml:"max_idleconn"`
	MaxIdleConnsPerHost int    `toml:"max_idleconn_perhost"`
	RetryTimes          int    `toml:"retry_times"`
	SlowTime            int    `toml:"slow_time"`
	EndpointsFrom       string `toml:"endpoints_from"`
	ConsulName          string `toml:"consul_name"`
	LoadBalanceStat     bool   `toml:"loadbalance_stat"`
	DC                  string `toml:"dc,omitempty"`

	// checker config
	CheckInterval      upstreamconfig.Duration `toml:"check_interval"`
	UnHealthyThreshold uint32                  `toml:"check_unhealth_threshold"`
	HealthyThreshold   uint32                  `toml:"check_healthy_threshold"`

	// lb advance config
	LBPanicThreshold int        `toml:"lb_panic_threshold"`
	LBSubsetKeys     [][]string `toml:"lb_subset_selectors"`
	LBDefaultKeys    []string   `toml:"lb_default_keys"`

	// detector config
	DetectInterval             upstreamconfig.Duration `toml:"detect_interval"`
	BaseEjectionDuration       upstreamconfig.Duration `toml:"base_ejection_duration"`
	ConsecutiveError           uint64                  `toml:"consecutive_error"`
	ConsecutiveConnectionError uint64                  `toml:"consecutive_connect_error"`
	MaxEjectionPercent         uint64                  `toml:"max_ejection_percent"`
	SuccessRateMinHosts        uint64                  `toml:"success_rate_min_hosts"`
	SuccessRateRequestVolume   uint64                  `toml:"success_rate_request_volume"`
	SuccessRateStdevFactor     float64                 `toml:"success_rate_stdev_factor"`
	Cluster                    upstreamconfig.Cluster

	Resource map[string]Resource `toml:"resource"` // 当用于http服务调用时, key=uri; 当用于rpc服务调用时, key=func_name; * 通配置
}

// 框架配置
type frameworkConfig struct {
	Server struct {
		ServiceName string   `toml:"service_name"`
		Port        int      `toml:"port"`
		Tags        []string `toml:"server_tags"`

		TCP struct {
			IdleTimeout      int `toml:"idle_timeout"`
			KeepliveInterval int `toml:"keeplive_interval"`
		} `toml:"tcp"`

		HTTP struct {
			Location     string `toml:"location"`
			LogResponse  string `toml:"logResponse"`
			ReadTimeout  int    `toml:"read_timeout"`  // 单位s,默认60s
			WriteTimeout int    `toml:"write_timeout"` // 单位s,默认60s
			IdleTimeout  int    `toml:"idle_timeout"`  // 单位s,默认90s
		} `toml:"http"`

		Breaker map[string]breaker.BreakerConfig   `toml:"breaker"` // 当用于http服务调用时, key=uri; 当用于rpc服务调用时, key=func_name; * 通配置
		Limiter map[string]ratelimit.LimiterConfig `toml:"limiter"` // // 当用于http服务调用时, key=app_name.service_name.uri; 当用于rpc服务调用时, key=app_name.service_name.func_name; * 通配置

		DefaultCircuit DefaultCircuit `toml:"default_circuit"`

		RecoverPanic bool `toml:"recover_panic"`
	} `toml:"server"`

	Trace struct {
		Port    int  `toml:"port"`
		Disable bool `toml:"disable"`
	} `toml:"trace"`

	Monitor struct {
		AliveInterval int `toml:"alive_interval"`
	} `toml:"monitor"`

	Log struct {
		Level              string `toml:"level"`
		Rotate             string `toml:"rotate"`
		AccessRotate       string `toml:"access_rotate"`
		Accesslog          string `toml:"accesslog"`
		Businesslog        string `toml:"businesslog"`
		Serverlog          string `toml:"serverlog"`
		StatLog            string `toml:"statlog"`
		ErrorLog           string `toml:"errlog"`
		LogPath            string `toml:"logpath"`
		BalanceLogLevel    string `toml:"balance_log_level"`
		GenLogLevel        string `toml:"gen_log_level"`
		AccessLogOff       bool   `toml:"access_log_off"`
		BusinessLogOff     bool   `toml:"business_log_off"`
		RequestBodyLogOff  bool   `toml:"request_log_off"`
		RespBodyLogMaxSize int    `toml:"response_log_max_size"` // -1:不限制;默认1024字节;
		SuccStatCode       []int  `toml:"succ_stat_code"`
	} `toml:"log"`

	ServerClient        []ServerClient             `toml:"server_client"`
	KafkaConsume        []kafka.KafkaConsumeConfig `toml:"kafka_consume"`
	KafkaProducerClient []kafkaProducerItem        `toml:"kafka_producer_client"`
	Redis               []redisConfig              `toml:"redis"`
	Database            []sql.SQLGroupConfig       `toml:"database"`
	Circuit             []CircuitConfig            `toml:"circuit"`
	DataLog             JSONDataLogOption          `toml:"data_log"`

	// 配置远程优先生效,默认不生效
	ConfigLoad struct {
		RemoteEnable bool `toml:"remote_enable"`
	} `toml:"cfg_load"`
}

type JSONDataLogOption struct {
	Path     string `toml:"path"`
	Rotate   string `toml:"rotate"`
	TaskName string `toml:"task_name"`
}

type CircuitConfig struct {
	Type       string   `toml:"type"`
	Service    string   `toml:"service"`
	Resource   string   `toml:"resource"`
	End        string   `toml:"end"`
	Open       bool     `toml:"open"`
	Threshold  float64  `toml:"threshold"`
	Strategy   string   `toml:"strategy"`
	MinSamples int64    `toml:"minsamples"`
	RT         duration `toml:"rt"`
}

type kafkaProducerItem struct {
	kafka.KafkaProductConfig
	Required_Acks string `toml:"required_acks"` // old rpc-go
	Use_Sync      bool   `toml:"use_sync"`      // old rpc-go
}

// golang包中的redis是json格式,此处转为toml格式
type redisConfig struct {
	ServerName     string `toml:"server_name"`
	Addr           string `toml:"addr"`
	Password       string `toml:"password"`
	MaxIdle        int    `toml:"max_idle"`
	MaxActive      int    `toml:"max_active"`
	IdleTimeout    int    `toml:"idle_timeout"`
	ConnectTimeout int    `toml:"connect_timeout"`
	ReadTimeout    int    `toml:"read_timeout"`
	WriteTimeout   int    `toml:"write_timeout"`
	Database       int    `toml:"database"`
	SlowTime       int    `toml:"slow_time"`
	Retry          int    `toml:"retry"`
}

type ClientOption func(*ClientOptions)

type ClientOptions struct{}
