package client

import (
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
	"github.com/yunfeiyang1916/toolkit/framework/breaker"
	"github.com/yunfeiyang1916/toolkit/framework/log"
	"github.com/yunfeiyang1916/toolkit/framework/ratelimit"
	"github.com/yunfeiyang1916/toolkit/go-upstream/upstream"
)

type Options struct {
	logger              log.Kit
	tracer              opentracing.Tracer
	dialTimeout         time.Duration
	idleConnTimeout     time.Duration
	keepAliveTimeout    time.Duration
	keepAlivesDisable   bool
	requestTimeout      time.Duration
	slowTimeout         time.Duration
	retryTimes          int
	maxIdleConnsPerHost int
	maxIdleConns        int
	cluster             *upstream.Cluster
	client              *http.Client
	limiter             *ratelimit.Config
	breaker             *breaker.Config
	namespace           string
	localName           string
	serviceName         string // 包含app_name的下游service_name
	protoType           string
}

type Option func(*Options)

func newOptions(options ...Option) Options {
	v := Options{}
	for _, o := range options {
		o(&v)
	}

	if v.logger == nil {
		v.logger = log.NewKit(
			log.New("./logs/business.log"), // bus
			log.New("./logs/access.log"),   // acc
			log.Stdout(),                   // err
		)
		v.logger.A().SetRotateByHour()
		v.logger.B().SetRotateByHour()
	}
	if v.tracer == nil {
		cfg := jaegerconfig.Configuration{
			Sampler: &jaegerconfig.SamplerConfig{Type: jaeger.SamplerTypeRemote},
			Reporter: &jaegerconfig.ReporterConfig{
				LogSpans:            false,
				BufferFlushInterval: 1 * time.Second,
				LocalAgentHostPort:  "127.0.0.1:6831",
			},
		}
		sName := v.localName
		if len(sName) == 0 {
			sName = "http-client-default"
		}
		tracer, _, _ := cfg.New(sName)
		v.tracer = tracer
	}

	if v.dialTimeout == 0 {
		v.dialTimeout = defaultDialTimeout
	}
	if v.keepAliveTimeout == 0 {
		v.keepAliveTimeout = defaultKeepAliveTimeout
	}
	if v.requestTimeout == 0 {
		v.requestTimeout = defaultRequestTimeout
	}
	if v.idleConnTimeout == 0 {
		v.idleConnTimeout = defaultIdleConnTimeout
	}
	if v.maxIdleConns == 0 {
		v.maxIdleConns = defaultMaxIdleConns
	}
	if v.maxIdleConnsPerHost == 0 {
		v.maxIdleConnsPerHost = defaultMaxIdleConnsPerHost
	}
	if v.slowTimeout == 0 {
		v.slowTimeout = defaultSlowTimeout
	}
	return v
}

func Breaker(config *breaker.Config) Option {
	return func(o *Options) {
		o.breaker = config
	}
}

func Limiter(config *ratelimit.Config) Option {
	return func(o *Options) {
		o.limiter = config
	}
}

func Logger(logger log.Kit) Option {
	return func(o *Options) {
		o.logger = logger
	}
}

func Tracer(tracer opentracing.Tracer) Option {
	return func(o *Options) {
		if tracer != nil {
			o.tracer = tracer
		}
	}
}

func DialTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.dialTimeout = d
	}
}

func RequestTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.requestTimeout = d
	}
}

func IdleConnTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.idleConnTimeout = d
	}
}

func KeepAliveTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.keepAliveTimeout = d
	}
}

// if true, do not re-use of TCP connections
func KeepAlivesDisable(t bool) Option {
	return func(o *Options) {
		o.keepAlivesDisable = t
	}
}

func RetryTimes(d int) Option {
	return func(o *Options) {
		o.retryTimes = d
	}
}

func MaxIdleConnsPerHost(d int) Option {
	return func(o *Options) {
		o.maxIdleConnsPerHost = d
	}
}

func MaxIdleConns(d int) Option {
	return func(o *Options) {
		o.maxIdleConns = d
	}
}

func Cluster(clu *upstream.Cluster) Option {
	return func(o *Options) {
		if clu != nil {
			o.cluster = clu
		}
	}
}

func WithClient(client *http.Client) Option {
	return func(o *Options) {
		if client != nil {
			o.client = client
		}
	}
}

func Namespace(n string) Option {
	return func(o *Options) {
		o.namespace = n
	}
}

func LocalName(n string) Option {
	return func(o *Options) {
		o.localName = n
	}
}

func ServiceName(sn string) Option {
	return func(o *Options) {
		o.serviceName = sn
	}
}

func ProtoType(p string) Option {
	return func(o *Options) {
		o.protoType = p
	}
}

func SlowTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.slowTimeout = d
	}
}
