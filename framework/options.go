package framework

import (
	"github.com/opentracing/opentracing-go"
	"github.com/yunfeiyang1916/toolkit/framework/log"
)

type Option func(framework *Framework)

// Mode TODO
type Mode int

const (
	// Deprecated: Development func should not use anymore.
	Development Mode = iota // 0
	// Deprecated: Production func should not use anymore.
	Production // 1
)

// Deprecated: String func should not use anymore.
func (m *Mode) String() string {
	switch *m {
	case Development:
		return "development"
	case Production:
		return "production"
	default:
		return "unknown"
	}
}

// Deprecated: Kit func should not use anymore.
func Kit(kit log.Kit) Option {
	return func(o *Framework) {
		//o.Kit = kit
	}
}

// Deprecated: RunMode func should not use anymore.
func RunMode(mode Mode) Option {
	return func(o *Framework) {
		//o.RunMode = mode
	}
}

func Namespace(namespace string) Option {
	return func(o *Framework) {
		o.Namespace = namespace
	}
}

func Name(name string) Option {
	return func(o *Framework) {
		o.Name = name
	}
}

func App(app string) Option {
	return func(o *Framework) {
		o.App = app
	}
}

func Version(ver string) Option {
	return func(o *Framework) {
		o.Version = ver
	}
}

func Deps(deps string) Option {
	return func(o *Framework) {
		o.Deps = deps
	}
}

// Deprecated: Tracer func should not use anymore.
func Tracer(tracer opentracing.Tracer) Option {
	return func(o *Framework) {
		//o.Tracer = tracer
	}
}

func ConfigPath(path string) Option {
	return func(o *Framework) {
		o.ConfigPath = path
	}
}

func NamespaceDir(dir string) Option {
	return func(o *Framework) {
		o.namespaceDir = dir
	}
}

func ConfigMemory(mem []byte) Option {
	return func(o *Framework) {
		o.ConfigMemory = mem
	}
}

// Deprecated: ConsulAddr func should not use anymore.
// 可以使用用户输入: -consul-addr="127.0.0.1:8500" 或者环境变量: "CONSUL_ADDR"
func ConsulAddr(addr string) Option {
	return func(o *Framework) {
		//o.ConsulAddr = addr
	}
}

// Deprecated: TraceReportAddr func should not use anymore.
// 可以使用用户输入: -trace-addr="127.0.0.1:6831" 或者环境变量: "TRACE_ADDR"
func TraceReportAddr(addr string) Option {
	return func(o *Framework) {
		//o.TraceReportAddr = addr
	}
}
