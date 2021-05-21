package server

import (
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
	"github.com/yunfeiyang1916/toolkit/framework/breaker"
	"github.com/yunfeiyang1916/toolkit/framework/log"
	"github.com/yunfeiyang1916/toolkit/framework/ratelimit"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
)

const (
	HTTPReadTimeout  = 60 * time.Second
	HTTPWriteTimeout = 60 * time.Second
	HTTPIdleTimeout  = 90 * time.Second
	defaultBodySize  = 1024
)

type Options struct {
	logger             log.Kit
	tracer             opentracing.Tracer
	serviceName        string
	port               int
	readTimeout        time.Duration
	writeTimeout       time.Duration
	idleTimeout        time.Duration // server keep conn
	certFile           string
	keyFile            string
	tags               map[string]string
	manager            *registry.ServiceManager
	registry           registry.Backend
	breaker            *breaker.Config
	limiter            *ratelimit.Config
	reqBodyLogOff      bool
	respBodyLogMaxSize int
	recoverPanic       bool
}

type Option func(*Options)

func newOptions(options ...Option) Options {
	opts := Options{}
	for _, o := range options {
		o(&opts)
	}

	if opts.logger == nil {
		opts.logger = log.NewKit(
			log.New("./logs/business.log"), // bus
			log.New("./logs/access.log"),   // acc
			log.Stdout(),                   // err
		)
		opts.logger.A().SetRotateByHour()
		opts.logger.B().SetRotateByHour()
	}

	if opts.tracer == nil {
		cfg := jaegerconfig.Configuration{
			Sampler: &jaegerconfig.SamplerConfig{Type: jaeger.SamplerTypeRemote},
			Reporter: &jaegerconfig.ReporterConfig{
				LogSpans:            false,
				BufferFlushInterval: 1 * time.Second,
				LocalAgentHostPort:  "127.0.0.1:6831",
			},
		}
		sName := opts.serviceName
		if len(sName) == 0 {
			sName = "http-server-default"
		}
		tracer, _, _ := cfg.New(sName)
		opts.tracer = tracer
	}

	if opts.readTimeout == 0 {
		opts.readTimeout = HTTPReadTimeout
	}
	if opts.writeTimeout == 0 {
		opts.writeTimeout = HTTPWriteTimeout
	}
	if opts.idleTimeout == 0 {
		opts.idleTimeout = HTTPIdleTimeout
	}
	if opts.respBodyLogMaxSize == 0 {
		opts.respBodyLogMaxSize = defaultBodySize
	}
	return opts
}

func Limiter(lim *ratelimit.Config) Option {
	return func(o *Options) {
		o.limiter = lim
	}
}

func Breaker(brk *breaker.Config) Option {
	return func(o *Options) {
		o.breaker = brk
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

func Port(port int) Option {
	return func(o *Options) {
		o.port = port
	}
}

func Name(serviceName string) Option {
	return func(o *Options) {
		o.serviceName = serviceName
	}
}

// 从连接被接受(accept)到request body完全被读取(如果你不读取body，那么时间截止到读完header为止)
// 包括了TCP消耗的时间,读header时间
// 对于 https请求，ReadTimeout 包括了TLS握手的时间
func ReadTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.readTimeout = d
	}
}

// 从request header的读取结束开始，到response write结束为止 (也就是 ServeHTTP 方法的声明周期)
func WriteTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.writeTimeout = d
	}
}

func IdleTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.idleTimeout = d
	}
}

func CertFile(file string) Option {
	return func(o *Options) {
		o.certFile = file
	}
}

func KeyFile(file string) Option {
	return func(o *Options) {
		o.keyFile = file
	}
}

func Tags(tags map[string]string) Option {
	return func(o *Options) {
		o.tags = tags
	}
}

func Manager(re *registry.ServiceManager) Option {
	return func(o *Options) {
		o.manager = re
	}
}

func Registry(r registry.Backend) Option {
	return func(o *Options) {
		if r != nil {
			o.registry = r
		}
	}
}

// 关闭req body 打印
func RequestBodyLogOff(b bool) Option {
	return func(o *Options) {
		o.reqBodyLogOff = b
	}
}

// 控制resp body打印大小
func RespBodyLogMaxSize(size int) Option {
	return func(o *Options) {
		o.respBodyLogMaxSize = size
	}
}

func RecoverPanic(rp bool) Option {
	return func(o *Options) {
		o.recoverPanic = rp
	}
}
