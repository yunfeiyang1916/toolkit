package server

import (
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
	"github.com/yunfeiyang1916/toolkit/framework/breaker"
	"github.com/yunfeiyang1916/toolkit/framework/log"
	"github.com/yunfeiyang1916/toolkit/framework/ratelimit"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/codec"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
)

type Option func(*Options)

type Options struct {
	Codec    codec.Codec
	Address  string
	Name     string
	Tracer   opentracing.Tracer
	Kit      log.Kit
	Manager  *registry.ServiceManager
	Tags     map[string]string
	Registry registry.Backend
	Limiter  *ratelimit.Config
	Breaker  *breaker.Config
}

func newOptions(opt ...Option) Options {
	opts := Options{}
	for _, o := range opt {
		o(&opts)
	}

	if len(opts.Address) == 0 {
		opts.Address = "127.0.0.1:10000"
	}

	if len(opts.Name) == 0 {
		opts.Name = "rpc-server-default"
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
		sName := opts.Name
		tracer, _, _ := cfg.New(sName)
		opts.Tracer = tracer
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

func Registry(r registry.Backend) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

func Tags(tags map[string]string) Option {
	return func(o *Options) {
		o.Tags = tags
	}
}

func Manager(b *registry.ServiceManager) Option {
	return func(o *Options) {
		o.Manager = b
	}
}

func LoggerKit(kit log.Kit) Option {
	return func(o *Options) {
		o.Kit = kit
	}
}

func Tracer(tracer opentracing.Tracer) Option {
	return func(o *Options) {
		o.Tracer = tracer
	}
}

func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

// Address to bind to - host:port
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

// Codec to use to encode/decode requests for a given content type
func Codec(c codec.Codec) Option {
	return func(o *Options) {
		o.Codec = c
	}
}
