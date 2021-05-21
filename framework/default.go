package framework

import (
	"fmt"
	"io"
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/yunfeiyang1916/toolkit/framework/breaker"
	"github.com/yunfeiyang1916/toolkit/framework/config"
	httpclient "github.com/yunfeiyang1916/toolkit/framework/http/client"
	httpserver "github.com/yunfeiyang1916/toolkit/framework/http/server"
	"github.com/yunfeiyang1916/toolkit/framework/log"
	"github.com/yunfeiyang1916/toolkit/framework/ratelimit"
	rpcclient "github.com/yunfeiyang1916/toolkit/framework/rpc/client"
	rpcserver "github.com/yunfeiyang1916/toolkit/framework/rpc/server"
	"github.com/yunfeiyang1916/toolkit/kafka"
	"github.com/yunfeiyang1916/toolkit/redis"
	"github.com/yunfeiyang1916/toolkit/sql"
	"golang.org/x/net/context"
)

var (
	DefaultKit           log.Kit
	defaultTracer        opentracing.Tracer
	defaultTraceCloser   io.Closer = noopCloser{}
	defaultServerLimiter *ratelimit.Config
	defaultServerBreaker *breaker.Config
)

var Default = New()

func AddMockRedisClient(name string, client *redis.Redis) error {
	return Default.AddRedisClient(name, client)
}

func AddMockSqlClient(name string, client *sql.Group) error {
	return Default.AddSqlClient(name, client)
}

func AddMockHTTPClient(name string, client httpclient.Client) error {
	return Default.AddHTTPClient(name, client)
}

func Init(options ...Option) {
	Default.Init(options...)
}

func ServiceName() string {
	return Default.localAppServiceName
}

func RPCServer() rpcserver.Server {
	return Default.RPCServer()
}

func HTTPServer() httpserver.Server {
	return Default.HTTPServer()
}

func Shutdown() error {
	return Default.Shutdown()
}

func Config() *config.Namespace {
	return Default.Config()
}

func ConfigInstance() config.Config {
	return Default.ConfigInstance()
}

func ConfigInstanceCtx(ctx context.Context) config.Config {
	d, ok := FromContext(ctx)
	if ok {
		return d.ConfigInstance()
	}
	return Default.ConfigInstance()
}

func File(files ...string) error {
	return Default.File(files...)
}

func Remote(paths ...string) error {
	return Default.Remote(paths...)
}

func RemoteKV(path string) (string, error) {
	return Default.RemoteKV(path)
}

func WatchKV(path string) config.Watcher {
	return Default.WatchKV(path)
}

func WatchPrefix(prefix string) config.Watcher {
	return Default.WatchPrefix(prefix)
}

func RPCFactory(ctx context.Context, name string) rpcclient.Factory {
	d, ok := FromContext(ctx)
	if ok {
		return d.RPCFactory(name)
	}
	return Default.RPCFactory(name)
}

func HTTPClient(ctx context.Context, name string) httpclient.Client {
	d, ok := FromContext(ctx)
	if ok {
		return d.HTTPClient(name)
	}
	return Default.HTTPClient(name)
}

func RedisClient(ctx context.Context, name string) *redis.Redis {
	d, ok := FromContext(ctx)
	if ok {
		return d.RedisClient(name)
	}
	return Default.RedisClient(name)
}

func SQLClient(ctx context.Context, name string) *sql.Group {
	d, ok := FromContext(ctx)
	if ok {
		return d.SQLClient(name)
	}
	return Default.SQLClient(name)
}

func KafkaConsumeClient(ctx context.Context, consumeFrom string) *kafka.KafkaConsumeClient {
	d, ok := FromContext(ctx)
	if ok {
		return d.KafkaConsumeClient(consumeFrom)
	}
	return Default.KafkaConsumeClient(consumeFrom)
}

func KafkaProducerClient(ctx context.Context, producerTo string) *kafka.KafkaClient {
	d, ok := FromContext(ctx)
	if ok {
		return d.KafkaProducerClient(producerTo)
	}
	return Default.KafkaProducerClient(producerTo)
}

func SyncProducerClient(ctx context.Context, producerTo string) *kafka.KafkaSyncClient {
	d, ok := FromContext(ctx)
	if ok {
		return d.SyncProducerClient(producerTo)
	}
	return Default.SyncProducerClient(producerTo)
}

func HTTPDoRequest(ctx context.Context, serviceName string, req *http.Request) (*httpclient.Response, error) {
	r := httpclient.NewRequest(ctx).WithRequest(req)
	c := HTTPClient(ctx, serviceName)
	if c != nil {
		return c.Call(r)
	}
	return nil, fmt.Errorf("httpclient with %s not found", serviceName)
}

func HTTPDo(ctx context.Context, serviceName, method, uri string, ro *httpclient.RequestOption, body io.Reader) (*httpclient.Response, error) {
	r := httpclient.NewRequest(ctx)
	if ro == nil {
		ro = &httpclient.RequestOption{}
	}
	d, ok := FromContext(ctx)
	if ok && len(serviceName) > 0 {
		sc, err := d.FindServerClient(serviceName)
		if err != nil {
			return nil, err
		}
		ro = &httpclient.RequestOption{}
		ro.RetryTimes(sc.RetryTimes)
		ro.RequestTimeoutMS(sc.ReadTimeout)
		ro.SlowTimeoutMS(sc.SlowTime)
	}
	r.WithURL(uri).
		WithMethod(method).
		WithBody(body).
		WithOption(ro)

	c := HTTPClient(ctx, serviceName)
	if c != nil {
		return c.Call(r)
	}
	return nil, fmt.Errorf("httpclient with %s not found", serviceName)
}
