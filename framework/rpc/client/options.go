package client

import (
	"time"

	"github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"

	"github.com/opentracing/opentracing-go"
	"github.com/yunfeiyang1916/toolkit/framework/breaker"
	"github.com/yunfeiyang1916/toolkit/framework/log"
	"github.com/yunfeiyang1916/toolkit/framework/ratelimit"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/codec"
	"github.com/yunfeiyang1916/toolkit/go-upstream/upstream"
)

type Options struct {
	Retries   int
	Kit       log.Kit
	Tracer    opentracing.Tracer
	Codec     codec.Codec
	Name      string // 包含app_name的下游service_name
	SDName    string
	Slow      time.Duration
	Limiter   *ratelimit.Config
	Breaker   *breaker.Config
	Cluster   *upstream.Cluster
	Namespace string

	// Connection Pool
	PoolSize int
	PoolTTL  time.Duration
	// Transport Dial Timeout
	DialTimeout time.Duration

	// Default CallOptions
	CallOptions CallOptions

	// only for testing
	dialer dialer

	maxIdleConnsPerHost int
	maxIdleConns        int
	keepAlivesDisable   bool
}

type CallOptions struct {
	// Number of Call attempts
	Retries int
	// Request/Response timeout
	RequestTimeout time.Duration
}

func newOptions(options ...Option) Options {
	opts := Options{
		Retries:     0,
		Kit:         nil,
		Tracer:      opentracing.NoopTracer{},
		PoolSize:    DefaultPoolSize,
		PoolTTL:     DefaultPoolTTL,
		Codec:       codec.NewProtoCodec(),
		DialTimeout: DefaultDialTimeout,
		Slow:        time.Millisecond * 30,
		CallOptions: CallOptions{
			Retries:        DefaultRetries,
			RequestTimeout: DefaultRequestTimeout,
		},
	}
	opts.dialer = defaultDialer{opts}

	for _, o := range options {
		o(&opts)
	}

	if opts.maxIdleConns == 0 {
		opts.maxIdleConns = 100
	}

	if opts.maxIdleConnsPerHost == 0 {
		opts.maxIdleConnsPerHost = 100
	}
	if opts.Kit == nil {
		kit := log.NewKit(
			log.New("./logs/business.log"),
			log.New("./logs/access.log"),
			log.Stdout(),
		)
		kit.A().SetRotateByHour()
		kit.B().SetRotateByHour()
		opts.Kit = kit
	}
	if opts.Tracer == nil {
		cfg := jaegerconfig.Configuration{
			Sampler: &jaegerconfig.SamplerConfig{Type: jaeger.SamplerTypeRemote},
			Reporter: &jaegerconfig.ReporterConfig{
				LogSpans:            false,
				BufferFlushInterval: 1 * time.Second,
				LocalAgentHostPort:  "127.0.0.1:6831",
			},
		}
		sName := opts.SDName
		if len(sName) == 0 {
			sName = "rpc-client-default"
		}
		tracer, _, _ := cfg.New(sName)
		opts.Tracer = tracer
	}
	return opts
}

func Breaker(config *breaker.Config) Option {
	return func(o *Options) {
		o.Breaker = config
	}
}

func Limiter(config *ratelimit.Config) Option {
	return func(o *Options) {
		o.Limiter = config
	}
}

func Slow(slow time.Duration) Option {
	return func(o *Options) {
		o.Slow = slow
	}
}

func SDName(n string) Option {
	return func(o *Options) {
		o.SDName = n
	}
}

func Namespace(n string) Option {
	return func(o *Options) {
		o.Namespace = n
	}
}

func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

// Logger sets the logger
func Kit(logger log.Kit) Option {
	return func(o *Options) {
		o.Kit = logger
	}
}

func Cluster(cluster *upstream.Cluster) Option {
	return func(o *Options) {
		o.Cluster = cluster
	}
}

// Codec sets the codec
func Codec(codec codec.Codec) Option {
	return func(o *Options) {
		o.Codec = codec
	}
}

// Tracer sets the opentracing tracer
func Tracer(tracer opentracing.Tracer) Option {
	return func(o *Options) {
		o.Tracer = tracer
	}
}

// PoolSize sets the connection pool size
func PoolSize(d int) Option {
	return func(o *Options) {
		o.PoolSize = d
	}
}

// PoolSize sets the connection pool size
func PoolTTL(d time.Duration) Option {
	return func(o *Options) {
		o.PoolTTL = d
	}
}

// Transport dial timeout
func DialTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.DialTimeout = d
	}
}

// Number of retries when making the request.
// Should this be a Call Option?
func Retries(i int) Option {
	return func(o *Options) {
		o.CallOptions.Retries = i
	}
}

// The request timeout.
// Should this be a Call Option?
func RequestTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.CallOptions.RequestTimeout = d
	}
}

// WithRetries is a CallOption which overrides that which
// set in Options.CallOptions
func WithRetries(i int) CallOption {
	return func(o *CallOptions) {
		o.Retries = i
	}
}

// WithRequestTimeout is a CallOption which overrides that which
// set in Options.CallOptions
func WithRequestTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.RequestTimeout = d
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

func KeepAlivesDisable(t bool) Option {
	return func(o *Options) {
		o.keepAlivesDisable = t
	}
}
