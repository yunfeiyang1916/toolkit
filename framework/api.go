package framework

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/yunfeiyang1916/toolkit/ecode"
	"github.com/yunfeiyang1916/toolkit/framework/breaker"
	"github.com/yunfeiyang1916/toolkit/framework/config"
	httpclient "github.com/yunfeiyang1916/toolkit/framework/http/client"
	httpserver "github.com/yunfeiyang1916/toolkit/framework/http/server"
	"github.com/yunfeiyang1916/toolkit/framework/ratelimit"
	rpcclient "github.com/yunfeiyang1916/toolkit/framework/rpc/client"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/codec"
	rpcserver "github.com/yunfeiyang1916/toolkit/framework/rpc/server"
	dutils "github.com/yunfeiyang1916/toolkit/framework/utils"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
	"github.com/yunfeiyang1916/toolkit/kafka"
	"github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/redis"
	"github.com/yunfeiyang1916/toolkit/sql"
)

func (d *Framework) Shutdown() error {
	// fixme:will be blocked
	// for _, client := range d.consumeClientMap {
	// 	result = multierror.Append(result, client.Close())
	// }

	result := d.shutdown()

	_ = defaultTraceCloser.Close()
	logging.Sync()

	return result
}

func (d *Framework) shutdown() error {
	var result error
	d.producerClients.Range(func(key, value interface{}) bool {
		switch value := value.(type) {
		case *kafka.KafkaClient:
			result = multierror.Append(result, value.Close())
		case *kafka.KafkaSyncClient:
			result = multierror.Append(result, value.Close())
		}
		return true
	})
	return result
}

func (d *Framework) Config() *config.Namespace {
	namespace := d.Namespace
	if n, ok := d.namespaceConfig.Load(namespace); ok {
		return n.(*config.Namespace)
	}
	prefix := getRegistryKVPath(d.localAppServiceName)
	if len(namespace) > 0 {
		prefix = path.Join(prefix, "app")
	}
	n, _ := d.namespaceConfig.LoadOrStore(
		namespace,
		config.NewNamespace(prefix).With(namespace),
	)
	return n.(*config.Namespace)
}

func (d *Framework) ConfigInstance() config.Config {
	return d.configInstance
}

func (d *Framework) File(files ...string) error {
	return d.configInstance.LoadFile(files...)

}
func (d *Framework) Remote(paths ...string) error {
	for _, p := range paths {
		if len(p) == 0 {
			continue
		}
		p = filepath.Join(getRegistryKVPath(d.localAppServiceName), p)
		err := d.configInstance.LoadPath(p, false, "toml")
		if err != nil {
			return err
		}
	}
	return nil
}

// path param is "/config", remote path is "/service_config/link/link.time.confignew/config"
func (d *Framework) RemoteKV(path string) (string, error) {
	p := filepath.Join(getRegistryKVPath(d.localAppServiceName), path)
	return config.RemoteKV(p)
}

func (d *Framework) WatchKV(path string) config.Watcher {
	prefix := getRegistryKVPath(d.localAppServiceName)
	p := filepath.Join(prefix, path)
	return config.WatchKV(p, prefix)
}

func (d *Framework) WatchPrefix(path string) config.Watcher {
	p := filepath.Join(getRegistryKVPath(d.localAppServiceName), path)
	return config.WatchPrefix(p)
}

func (d *Framework) InjectServerClient(sc ServerClient) {
	if atomic.LoadInt32(&d.pendingServerClientTaskDone) == 0 {
		d.pendingServerClientLock.Lock()
		defer d.pendingServerClientLock.Unlock()
		if atomic.LoadInt32(&d.pendingServerClientTaskDone) == 0 {
			d.pendingServerClientTask = append(d.pendingServerClientTask, sc)
			return
		}
	}
	d.injectServerClient(sc)
}

// 对于以下方法中的service参数说明:
// 如果对应的server_client配置了app_name选项,则需要调用方保证service参数带上app_name前缀
// 如果没有配置,则保持原有逻辑,	service参数不用改动
func (d *Framework) FindServerClient(service string) (ServerClient, error) {
	if value, ok := d.serverClientMap.Load(service); ok {
		sc := value.(ServerClient)
		return sc, nil
	}
	return ServerClient{}, fmt.Errorf("client config for %s not exist", service)
}

func (d *Framework) ServiceClientWithApp(appName, serviceName string) (ServerClient, error) {
	appServiceName := dutils.MakeAppServiceName(appName, serviceName)
	return d.FindServerClient(appServiceName)
}

// RPC create a new rpc client instance, default use http protocol.
func (d *Framework) RPCFactory(name string) rpcclient.Factory {
	if c, ok := d.rpcClientMap.Load(name); ok {
		return c.(rpcclient.Factory)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if c, ok := d.rpcClientMap.Load(name); ok {
		return c.(rpcclient.Factory)
	}

	sc, err := d.FindServerClient(name)
	if err != nil {
		fmt.Printf("namespace %s rpcclient %s not exist, err %v\n", d.Namespace, name, err)
		logging.GenLogf("namespace %s rpcclient %s not exist, err %v", d.Namespace, name, err)
		return nil
	}
	sName := sc.ServiceName
	if sc.APPName != nil {
		if len(*sc.APPName) > 0 && *sc.APPName != _app {
			sName = fmt.Sprintf("%s.%s", *sc.APPName, sc.ServiceName)
		}
	}

	var clusterName string
	if sc.APPName != nil {
		clusterName = fmt.Sprintf("%s-http", sName)
	} else {
		clusterName = fmt.Sprintf("%s-http", dutils.MakeAppServiceName(d.App, sc.ServiceName))
	}

	client := rpcclient.HFactory(
		rpcclient.Cluster(d.Clusters.Cluster(clusterName)),
		rpcclient.Kit(DefaultKit),
		rpcclient.Tracer(defaultTracer),
		rpcclient.Codec(codec.NewJSONCodec()),
		rpcclient.MaxIdleConns(sc.MaxIdleConns),
		rpcclient.MaxIdleConnsPerHost(sc.MaxIdleConnsPerHost),
		rpcclient.DialTimeout(time.Duration(sc.ConnectTimeout)*time.Millisecond),
		rpcclient.Retries(sc.RetryTimes),
		rpcclient.RequestTimeout(time.Duration(sc.ReadTimeout)*time.Millisecond),
		rpcclient.Slow(time.Duration(sc.SlowTime)*time.Millisecond),
		rpcclient.SDName(d.localAppServiceName), // 本地服务发现名
		rpcclient.Name(sName),                   // 下游服务发现名
		rpcclient.Namespace(sc.Namespace),       // 下游所属namespace
		rpcclient.Limiter(ratelimit.NewConfig(getClientLimiterConfig(sc.Namespace, sc))),
		rpcclient.Breaker(breaker.NewConfig(getClientBreakerConfig(sc.Namespace, sc))),
	)
	d.rpcClientMap.Store(name, client)
	return client
}

func (d *Framework) RPCServer() rpcserver.Server {
	port := d.config.Server.Port
	if port == 0 {
		panic("server port is 0")
	}

	server := rpcserver.BothServer(
		d.localAppServiceName,
		port,
		rpcserver.Name(d.localAppServiceName),
		rpcserver.Tracer(defaultTracer),
		rpcserver.LoggerKit(DefaultKit),
		rpcserver.Tags(getServiceTags(d.config.Server.Tags)),
		rpcserver.Manager(d.Manager),
		rpcserver.Registry(registry.Default),
		rpcserver.Limiter(defaultServerLimiter),
		rpcserver.Breaker(defaultServerBreaker),
	)
	return server
}

func (d *Framework) HTTPClient(name string) httpclient.Client {
	if len(name) == 0 {
		if c, ok := d.httpClientMap.Load("default"); ok {
			return c.(httpclient.Client)
		}
		d.mu.Lock()
		defer d.mu.Unlock()

		if c, ok := d.httpClientMap.Load("default"); ok {
			return c.(httpclient.Client)
		}
		c := httpclient.NewClient(
			httpclient.Tracer(defaultTracer),
			httpclient.Logger(DefaultKit),
			httpclient.LocalName(d.localAppServiceName),
		)
		d.httpClientMap.Store("default", c)
		return c
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	sc, err := d.FindServerClient(name)
	if err != nil {
		fmt.Printf("namespace %s httpclient %s not exist, err %v\n", d.Namespace, name, err)
		logging.GenLogf("namespace %s httpclient %s not exist, err %v", d.Namespace, name, err)
		return nil
	}
	if sc.ProtoType == "" || sc.ProtoType == "rpc" {
		sc.ProtoType = "http"
	}
	sName := sc.ServiceName
	if sc.APPName != nil {
		if len(*sc.APPName) > 0 && *sc.APPName != _app {
			sName = fmt.Sprintf("%s.%s", *sc.APPName, sc.ServiceName)
		}
	}
	if v, ok := d.httpClientMap.Load(sName); ok {
		return v.(httpclient.Client)
	}
	var clusterName string
	if sc.APPName != nil {
		clusterName = fmt.Sprintf("%s-%s", sName, sc.ProtoType)
	} else {
		clusterName = fmt.Sprintf("%s-%s", dutils.MakeAppServiceName(d.App, sc.ServiceName), sc.ProtoType)
	}

	client := httpclient.NewClient(
		httpclient.Cluster(d.Clusters.Cluster(clusterName)),
		httpclient.Logger(DefaultKit),
		httpclient.Tracer(defaultTracer),
		httpclient.MaxIdleConns(sc.MaxIdleConns),
		httpclient.MaxIdleConnsPerHost(sc.MaxIdleConnsPerHost),
		httpclient.RetryTimes(sc.RetryTimes),
		httpclient.DialTimeout(time.Duration(sc.ConnectTimeout)*time.Millisecond),
		httpclient.RequestTimeout(time.Duration(sc.ReadTimeout)*time.Millisecond),
		httpclient.SlowTimeout(time.Duration(sc.SlowTime)*time.Millisecond),
		httpclient.KeepAliveTimeout(time.Duration(sc.KeepaliveTimeout)*time.Millisecond),
		httpclient.LocalName(d.localAppServiceName), // 本地服务发现名
		httpclient.ServiceName(sName),               // 下游服务发现名
		httpclient.Namespace(sc.Namespace),          // 下游所属namespace
		httpclient.ProtoType(sc.ProtoType),
		httpclient.Limiter(ratelimit.NewConfig(getClientLimiterConfig(sc.Namespace, sc))),
		httpclient.Breaker(breaker.NewConfig(getClientBreakerConfig(sc.Namespace, sc))),
	)

	d.httpClientMap.Store(sName, client)
	return client
}

func (d *Framework) HTTPServer() httpserver.Server {
	httpServer := httpserver.NewServer(
		httpserver.Name(d.localAppServiceName),
		httpserver.Port(d.config.Server.Port),
		httpserver.Tracer(defaultTracer),
		httpserver.Logger(DefaultKit),
		httpserver.Tags(getServiceTags(d.config.Server.Tags)),
		httpserver.Manager(d.Manager),
		httpserver.Registry(registry.Default),
		httpserver.Limiter(defaultServerLimiter),
		httpserver.Breaker(defaultServerBreaker),
		httpserver.RequestBodyLogOff(d.config.Log.RequestBodyLogOff),
		httpserver.RespBodyLogMaxSize(d.config.Log.RespBodyLogMaxSize),
		httpserver.RecoverPanic(d.config.Server.RecoverPanic),
		httpserver.ReadTimeout(time.Duration(d.config.Server.HTTP.ReadTimeout)*time.Second),
		httpserver.WriteTimeout(time.Duration(d.config.Server.HTTP.WriteTimeout)*time.Second),
		httpserver.IdleTimeout(time.Duration(d.config.Server.HTTP.IdleTimeout)*time.Second),
	)
	// default export API
	{
		// Register a router to get pprof port of this application
		httpServer.ANY(_pprofURI, func(c *httpserver.Context) {
			c.JSON(map[string]interface{}{"port": d.config.Trace.Port}, nil)
		})
		httpServer.POST("/debug/set", func(c *httpserver.Context) {
			var r struct {
				LogLevel string `json:"log_level"`
			}
			buf, err := ioutil.ReadAll(c.Request.Body)
			if err != nil {
				c.JSON(nil, ecode.ServerErr)
				return
			}
			err = json.Unmarshal(buf, &r)
			if err != nil {
				c.JSON(nil, ecode.ServerErr)
				return
			}
			logging.SetLevelByString(r.LogLevel)
			c.JSON(nil, ecode.OK)
		})
	}
	return httpServer
}

func (d *Framework) RedisClient(name string) *redis.Redis {
	if client, ok := d.redisClients.Load(name); ok {
		if v, ok1 := client.(*redis.Redis); ok1 {
			return v
		}
	}
	fmt.Printf("namespace %s redis client for %s not exist\n", d.Namespace, name)
	logging.GenLogf("namespace %s redis client for %s not exist", d.Namespace, name)
	return nil
}

func (d *Framework) SQLClient(name string) *sql.Group {
	if client, ok := d.mysqlClients.Load(name); ok {
		if v, ok1 := client.(*sql.Group); ok1 {
			return v
		}
	}
	fmt.Printf("namespace %s mysql client for %s not exist\n", d.Namespace, name)
	logging.GenLogf("namespace %s mysql client for %s not exist", d.Namespace, name)
	return nil
}

func (d *Framework) KafkaConsumeClient(consumeFrom string) *kafka.KafkaConsumeClient {
	if client, ok := d.consumeClients.Load(consumeFrom); ok {
		if v, ok1 := client.(*kafka.KafkaConsumeClient); ok1 {
			return v
		}
	}
	fmt.Printf("namespace %s kafka consume client %s not exist\n", d.Namespace, consumeFrom)
	logging.GenLogf("namespace %s kafka consume client %s not exist", d.Namespace, consumeFrom)
	return nil
}

func (d *Framework) KafkaProducerClient(producerTo string) *kafka.KafkaClient {
	if client, ok := d.producerClients.Load(producerTo); ok {
		if v, ok := client.(*kafka.KafkaClient); ok {
			return v
		}
		fmt.Printf("namespace %s kafka producer %s type not match, should use SyncProducerClient()\n", d.Namespace, producerTo)
		logging.GenLogf("namespace %s kafka producer %s type not match, should use SyncProducerClient()", d.Namespace, producerTo)
		return nil
	}
	fmt.Printf("namespace %s kafka producer client %s to not exist\n", d.Namespace, producerTo)
	logging.GenLogf("namespace %s kafka producer client %s to not exist", d.Namespace, producerTo)
	return nil
}

func (d *Framework) SyncProducerClient(producerTo string) *kafka.KafkaSyncClient {
	if client, ok := d.producerClients.Load(producerTo); ok {
		if v, ok := client.(*kafka.KafkaSyncClient); ok {
			return v
		}
		fmt.Printf("namespace %s kafka sync producer %s type not match, should use KafkaProducerClient()\n", d.Namespace, producerTo)
		logging.GenLogf("namespace %s kafka sync producer %s type not match, should use KafkaProducerClient()", d.Namespace, producerTo)
		return nil
	}
	fmt.Printf("namespace %s kafka sync producer client %s not exist\n", d.Namespace, producerTo)
	logging.GenLogf("namespace %s kafka sync producer client %s not exist", d.Namespace, producerTo)
	return nil
}

func (d *Framework) InitKafkaProducer(kpcList []kafka.KafkaProductConfig) error {
	for _, item := range kpcList {
		if _, ok := d.producerClients.Load(item.ProducerTo); ok {
			continue
		}
		if item.UseSync {
			client, err := kafka.NewSyncProducterClient(item)
			if err != nil {
				return err
			}
			// 忽略已存在的记录
			d.producerClients.LoadOrStore(item.ProducerTo, client)
		} else {
			client, err := kafka.NewKafkaClient(item)
			if err != nil {
				return err
			}
			d.producerClients.LoadOrStore(item.ProducerTo, client)
		}
	}
	return nil
}

func (d *Framework) InitKafkaConsume(kccList []kafka.KafkaConsumeConfig) error {
	for _, item := range kccList {
		if _, ok := d.consumeClients.Load(item.ConsumeFrom); ok {
			continue
		}
		client, err := kafka.NewKafkaConsumeClient(item)
		if err != nil {
			return err
		}
		d.consumeClients.LoadOrStore(item.ConsumeFrom, client)
	}
	return nil
}

func (d *Framework) InitRedisClient(rcList []redis.RedisConfig) error {
	for _, c := range rcList {
		if _, ok := d.redisClients.Load(c.ServerName); ok {
			continue
		}
		cc := c
		client, err := redis.NewRedis(&cc)
		if err != nil {
			return err
		}
		d.redisClients.LoadOrStore(cc.ServerName, client)
	}
	return nil
}

func (d *Framework) InitSqlClient(sqlList []sql.SQLGroupConfig) error {
	for _, c := range sqlList {
		if _, ok := d.mysqlClients.Load(c.Name); ok {
			continue
		}
		g, err := sql.NewGroup(c)
		if err != nil {
			return err
		}
		_ = sql.SQLGroupManager.Add(c.Name, g)
		d.mysqlClients.LoadOrStore(c.Name, g)
	}
	return nil
}

func (d *Framework) AddSqlClient(name string, client *sql.Group) error {
	d.mysqlClients.LoadOrStore(name, client)
	return nil
}

func (d *Framework) AddRedisClient(name string, client *redis.Redis) error {
	d.redisClients.LoadOrStore(name, client)
	return nil
}

func (d *Framework) AddSyncKafkaClient(name string, client *kafka.KafkaSyncClient) error {
	d.producerClients.LoadOrStore(name, client)
	return nil
}

func (d *Framework) AddAsyncKafkaClient(name string, client *kafka.KafkaClient) error {
	d.producerClients.LoadOrStore(name, client)
	return nil
}

func (d *Framework) AddHTTPClient(name string, client httpclient.Client) error {
	d.httpClientMap.LoadOrStore(name, client)
	d.serverClientMap.LoadOrStore(name, ServerClient{ServiceName: name})
	return nil
}

func (d *Framework) initBreaker() {
	defaultServerBreaker.AddConfig(getServerBreakerConfig(d.Namespace, d.config))
	breaker.InitDefaultConfig(getDefaultBreakerConfig(d.Namespace, d.config.Server.DefaultCircuit))
}

func (d *Framework) initLimiter() {
	defaultServerLimiter.AddConfig(getServerLimiterConfig(d.Namespace, d.config))
	ratelimit.InitDefaultConfig(getDefaultLimiterConfig(d.Namespace, d.config.Server.DefaultCircuit))
}
