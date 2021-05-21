package framework

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
	"github.com/yunfeiyang1916/toolkit/framework/breaker"
	"github.com/yunfeiyang1916/toolkit/framework/config"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/json"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/toml"
	"github.com/yunfeiyang1916/toolkit/framework/config/reader"
	ns "github.com/yunfeiyang1916/toolkit/framework/internal/kit/namespace"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/sd"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/tracing"
	"github.com/yunfeiyang1916/toolkit/framework/log"
	"github.com/yunfeiyang1916/toolkit/framework/ratelimit"
	dutils "github.com/yunfeiyang1916/toolkit/framework/utils"
	clusterconfig "github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry/consul"
	"github.com/yunfeiyang1916/toolkit/go-upstream/upstream"
	"github.com/yunfeiyang1916/toolkit/kafka"
	"github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/metrics"
	"github.com/yunfeiyang1916/toolkit/redis"
	"github.com/yunfeiyang1916/toolkit/rolling"
	"github.com/yunfeiyang1916/toolkit/utils"
	"golang.org/x/net/trace"
)

type noopCloser struct{}

func (n noopCloser) Close() error {
	return nil
}

func readFile(name string) string {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		return ""
	}
	return string(b)
}

func getServiceTags(tags []string) map[string]string {
	serviceTags := make(map[string]string)
	for _, t := range tags {
		kvs := strings.SplitN(t, "=", 2)
		if len(kvs) > 1 {
			serviceTags[strings.TrimSpace(kvs[0])] = strings.TrimSpace(kvs[1])
		}
	}
	return serviceTags
}

func getRegistryKVPath(name string) string {
	namespace := strings.Split(name, ".")[0]
	return path.Join("/service_config", namespace, name)
}

func getClientBreakerConfig(namespace string, sc ServerClient) []breaker.BreakerConfig {
	var vals []breaker.BreakerConfig
	/*
		"*": {}
		"/api/sms/send": {}
	*/
	for k, v := range sc.Resource {
		serviceName := sc.ServiceName
		if sc.APPName != nil && *sc.APPName != _app {
			serviceName = dutils.MakeAppServiceName(*sc.APPName, sc.ServiceName)
		}
		vals = append(vals, breaker.BreakerConfig{
			Name:                      breaker.GetClientBreakerName(namespace, serviceName, k),
			ErrorPercentThreshold:     v.ErrorPercentThreshold,
			ConsecutiveErrorThreshold: v.ConsecutiveErrorThreshold,
			MinSamples:                v.MinSamples,
		})
	}
	return vals
}

func getServerBreakerConfig(namespace string, c frameworkConfig) []breaker.BreakerConfig {
	var vals []breaker.BreakerConfig
	for k, v := range c.Server.Breaker {
		/*
			"/api/sms/send" : {} ===> namespace.server./api/sms/send
			"*": {} ===> namespace.server.*
		*/
		vals = append(vals, breaker.BreakerConfig{
			Name:                      breaker.GetServerBreakerName(namespace, k),
			ErrorPercentThreshold:     v.ErrorPercentThreshold,
			ConsecutiveErrorThreshold: v.ConsecutiveErrorThreshold,
			MinSamples:                v.MinSamples,
		})
	}
	return vals
}

func getClientLimiterConfig(namespace string, sc ServerClient) []ratelimit.LimiterConfig {
	var vals []ratelimit.LimiterConfig
	for k, v := range sc.Resource {
		serviceName := sc.ServiceName
		if sc.APPName != nil && *sc.APPName != _app {
			serviceName = dutils.MakeAppServiceName(*sc.APPName, sc.ServiceName)
		}

		vals = append(vals, ratelimit.LimiterConfig{
			Name:   ratelimit.GetClientLimiterName(namespace, serviceName, k),
			Limits: v.Limits,
			Open:   v.Open,
		})
	}
	return vals
}

func getServerLimiterConfig(namespace string, c frameworkConfig) []ratelimit.LimiterConfig {
	var vals []ratelimit.LimiterConfig
	for k, v := range c.Server.Limiter {
		ss := strings.Split(k, "@")
		if len(ss) != 2 {
			continue
		}
		vals = append(vals, ratelimit.LimiterConfig{
			Name:   ratelimit.GetServerLimiterName(namespace, ss[0], ss[1]),
			Limits: v.Limits,
			Open:   v.Open,
		})
	}
	return vals
}

func getBreakerConfig(namespace string, c frameworkConfig) []breaker.BreakerConfig {
	var configs []breaker.BreakerConfig
	configs = append(configs, getServerBreakerConfig(namespace, c)...)
	for _, v := range c.ServerClient {
		configs = append(configs, getClientBreakerConfig(v.Namespace, v)...)
	}
	configs = append(configs, getDefaultBreakerConfig(namespace, c.Server.DefaultCircuit)...)
	return configs
}

func getLimiterConfig(namespace string, c frameworkConfig) []ratelimit.LimiterConfig {
	var configs []ratelimit.LimiterConfig
	configs = append(configs, getServerLimiterConfig(namespace, c)...)
	for _, v := range c.ServerClient {
		configs = append(configs, getClientLimiterConfig(v.Namespace, v)...)
	}
	configs = append(configs, getDefaultLimiterConfig(namespace, c.Server.DefaultCircuit)...)
	return configs
}

func getDefaultBreakerConfig(namespace string, dc DefaultCircuit) []breaker.BreakerConfig {
	var vals []breaker.BreakerConfig
	vals = append(vals, breaker.BreakerConfig{
		Name:                      breaker.GetDefaultServerBreakerName(namespace),
		ErrorPercentThreshold:     dc.Server.ErrorPercentThreshold,
		ConsecutiveErrorThreshold: dc.Server.ConsecutiveErrorThreshold,
		MinSamples:                dc.Server.MinSamples,
		Break:                     dc.Server.Break,
	})
	vals = append(vals, breaker.BreakerConfig{
		Name:                      breaker.GetDefaultClientBreakerName(namespace),
		ErrorPercentThreshold:     dc.Client.ErrorPercentThreshold,
		ConsecutiveErrorThreshold: dc.Client.ConsecutiveErrorThreshold,
		MinSamples:                dc.Client.MinSamples,
		Break:                     dc.Client.Break,
	})
	return vals
}

func getDefaultLimiterConfig(namespace string, dc DefaultCircuit) []ratelimit.LimiterConfig {
	var vals []ratelimit.LimiterConfig
	vals = append(vals, ratelimit.LimiterConfig{
		Name:   ratelimit.GetDefaultServerLimiterName(namespace),
		Limits: dc.Server.Limits,
		Open:   dc.Server.Open,
	})
	vals = append(vals, ratelimit.LimiterConfig{
		Name:   ratelimit.GetDefaultClientLimiterName(namespace),
		Limits: dc.Client.Limits,
		Open:   dc.Client.Open,
	})
	return vals
}

// serviceName初始化优先级:
// .discovery文件serviceName > 本地配置文件serviceName > 服务二进制名字serviceName
func (d *Framework) initName(c []byte) {
	// 部署系统文件serviceName
	d.Name = strings.TrimSpace(readFile(".discovery"))
	if len(d.Name) == 0 && len(c) > 0 {
		// 本地配置文件serviceName
		_ = toml.NewEncoder().Decode(c, &d.config)
		if len(d.config.Server.ServiceName) != 0 {
			d.Name = d.config.Server.ServiceName
		} else if len(os.Args) > 0 {
			// 服务二进制名字serviceName
			d.Name = filepath.Base(os.Args[0])
		}
	}
}

func (d *Framework) initLocalAppServiceName(c []byte) {
	d.initName(c)
	d.localAppServiceName = dutils.MakeAppServiceName(d.App, d.Name)
}

func (d *Framework) loadLocalConfig() []byte {
	if len(d.ConfigPath) == 0 {
		return nil
	}
	c := config.New()
	_ = c.LoadFile(d.ConfigPath)
	_ = c.Scan(&d.config)
	return c.Bytes()
}

func makeUploadPath(dir string, filename string) string {
	var remotePath string
	if len(dir) > 0 {
		appDir := filepath.Base(dir)
		dd := filepath.Clean(dir)
		filename = filepath.Clean(filename)
		appConfigPath := strings.TrimPrefix(strings.TrimPrefix(filename, dd), "/")
		if appDir == ".." || appDir == "." {
			appDir = ""
		}
		remotePath = filepath.Join(appDir, appConfigPath) // appdir/appname/config.toml
	} else {
		remotePath = filepath.Base(filename) // config.toml
	}
	return remotePath
}

func (d *Framework) uploadConfig(filename string) {
	dc, err := sd.GetDatacenter(consulAddr)
	if err != nil {
		return
	}

	content := readFile(filename)
	if len(content) == 0 {
		return
	}
	body := map[string]interface{}{}
	body["type"] = 0
	body["content"] = content
	body["service"] = d.localAppServiceName
	body["cluster"] = dc
	body["path"] = makeUploadPath(d.namespaceDir, filename)
	body["force"] = false

	b, _ := json.NewEncoder().Encode(body)
	respB, err := tracing.KVPut(b)
	logging.GenLogf("sync local config to consul, err %v, response %q", err, respB)
	if err != nil {
		return
	}
}

// 包级别全局公共资源,只需要初始化一次
func (d *Framework) initDefaultOnce() {
	initOnce.Do(func() {
		d.initLogger()
		d.initGolangTrace()
		d.initStat()
		if err := d.initTracer(); err != nil {
			panic(err)
		}

		// breaker limiter watcher init once
		remotePath := getRegistryKVPath(d.localAppServiceName)
		initConfigWatcher(remotePath)
		defaultServerBreaker = breaker.NewConfig(nil)
		defaultServerLimiter = ratelimit.NewConfig(nil)
	})
}

func (d *Framework) initConfigInstance() error {
	if len(d.ConfigMemory) > 0 {
		// 加载配置
		c := d.Config().GetMemoryD(d.ConfigMemory)
		if err := c.Scan(&d.config); err != nil {
			return err
		}

		d.configInstance = c
	} else {
		// 加载配置
		c := d.Config().GetD("config.toml", d.ConfigPath, rfc.status())
		if err := c.Scan(&d.config); err != nil {
			return err
		}

		d.configInstance = c

		// 上传本地配置到kae,该配置指框架约定的config.toml,包含app子路径下的config.toml
		d.uploadConfig(d.ConfigPath)
	}

	return nil
}

//nolint:unused
func (d *Framework) initConsulBackend() {
	registry.Default, _ = consul.NewBackend(&clusterconfig.Consul{Addr: consulAddr, Scheme: "http", Logger: logging.Log(logging.GenLoggerName)})
}

func (d *Framework) initTracer() error {
	if defaultTracer == nil {
		cfg := jaegerconfig.Configuration{
			// SamplingServerURL: "http://localhost:5778/sampling"
			Sampler: &jaegerconfig.SamplerConfig{Type: jaeger.SamplerTypeRemote},
			Reporter: &jaegerconfig.ReporterConfig{
				LogSpans:            false,
				BufferFlushInterval: 1 * time.Second,
				// LocalAgentHostPort:  d.TraceReportAddr,
				LocalAgentHostPort: traceReportAddr,
			},
		}
		tracer, closer, err := cfg.New(d.localAppServiceName)
		if err != nil {
			return err
		}
		defaultTraceCloser = closer
		defaultTracer = tracer
	}
	if defaultTracer != nil {
		opentracing.SetGlobalTracer(defaultTracer)
	}
	return nil
}

func (d *Framework) initLogger() {
	if len(d.config.Log.LogPath) == 0 {
		d.config.Log.LogPath = "logs"
	}
	d.LogDir = d.config.Log.LogPath

	// Init common logger
	logging.InitCommonLog(logging.CommonLogConfig{
		Pathprefix:      d.config.Log.LogPath,
		Rotate:          d.config.Log.Rotate,
		GenLogLevel:     d.config.Log.GenLogLevel,
		BalanceLogLevel: d.config.Log.BalanceLogLevel,
	})

	// upstream logger
	upstream.SetLogger(logging.Log(logging.BalanceLoggerName))

	if d.config.Log.Rotate == LOG_ROTATE_DAY {
		logging.SetRotateByDay()
	} else {
		logging.SetRotateByHour()
	}
	if len(d.config.Log.Level) > 0 {
		logging.SetLevelByString(d.config.Log.Level)
	} else {
		logging.SetLevelByString(d.LogLevel)
	}
	// will init debug info error logger inside
	logging.SetOutputPath(d.LogDir)

	if d.config.DataLog.Path != "" {
		var rotate rolling.RollingFormat
		switch d.config.DataLog.Rotate {
		case LOG_ROTATE_HOUR:
			rotate = rolling.HourlyRolling
		case LOG_ROTATE_DAY:
			rotate = rolling.DailyRolling
		case LOG_ROTATE_MONTH:
			rotate = rolling.MinutelyRolling
		default:
			rotate = rolling.DailyRolling
		}
		name := d.localAppServiceName
		if d.config.DataLog.TaskName != "" {
			name = d.config.DataLog.TaskName
		}
		_ = logging.InitDataWithKey(d.config.DataLog.Path, rotate, name)
	}
	// internal logger
	rotateType := d.config.Log.Rotate
	var blog, alog *logging.Logger
	if !d.config.Log.AccessLogOff {
		alog = log.New(filepath.Join(d.LogDir, "access.log"))
		if rotateType == "day" {
			alog.SetRotateByDay()
		}
	}
	if !d.config.Log.BusinessLogOff {
		blog = log.New(filepath.Join(d.LogDir, "business.log"))
		if rotateType == "day" {
			blog.SetRotateByDay()
		}
	}
	// FIXME: should remove
	elog := log.New(filepath.Join(d.LogDir, "error.log"))
	elog.SetLevelByString("error")
	if rotateType == "day" {
		elog.SetRotateByDay()
	}

	if DefaultKit == nil {
		DefaultKit = log.NewKit(blog, alog, elog)
	}
}

// 如果设置了app_name则用app_name+service_name,如果没有则保持原有逻辑用service_name
// 此处逻辑保证注册与获取时service_name是一致的
func (d *Framework) injectServerClient(sc ServerClient) {
	sName := sc.ServiceName
	if sc.APPName != nil {
		if len(*sc.APPName) > 0 && *sc.APPName != _app {
			sName = dutils.MakeAppServiceName(*sc.APPName, sc.ServiceName)
		}
	}
	if _, ok := d.serverClientMap.Load(sName); ok {
		return
	}
	cluster := d.makeCluster(sName, sc)
	if err := d.Clusters.InitService(cluster); err != nil {
		panic(err)
	}
	sc.Cluster = cluster
	d.serverClientMap.Store(sName, sc)
}

func (d *Framework) makeCluster(sName string, sc ServerClient) clusterconfig.Cluster {
	cluster := clusterconfig.NewCluster()
	if sc.ProtoType == "" || sc.ProtoType == "rpc" {
		sc.ProtoType = "http"
	}
	cluster.Name = fmt.Sprintf("%s-%s", sName, sc.ProtoType)
	if sc.APPName == nil { // 原有逻辑:使用本地环境的app_name
		cluster.Name = fmt.Sprintf("%s-%s", dutils.MakeAppServiceName(d.App, sc.ServiceName), sc.ProtoType)
	}
	cluster.StaticEndpoints = sc.Ipport
	if len(sc.Ipport) != 0 {
		// add fallback port
		var fallbackPort = ""
		if sc.ProtoType == "http" {
			fallbackPort = ":80"
		} else if sc.ProtoType == "https" {
			fallbackPort = ":443"
		}
		staticIPPorts := strings.Split(sc.Ipport, ",")
		for i := range staticIPPorts {
			_, _, err := net.SplitHostPort(staticIPPorts[i])
			if err != nil {
				if strings.Contains(err.Error(), "missing port") {
					staticIPPorts[i] = staticIPPorts[i] + fallbackPort
				}
			}
		}
		cluster.StaticEndpoints = strings.Join(staticIPPorts, ",")
	}
	cluster.Proto = sc.ProtoType
	cluster.LBType = sc.Balancetype
	cluster.EndpointsFrom = sc.EndpointsFrom
	cluster.CheckInterval = sc.CheckInterval
	cluster.UnHealthyThreshold = sc.UnHealthyThreshold
	cluster.HealthyThreshold = sc.HealthyThreshold
	cluster.LBPanicThreshold = sc.LBPanicThreshold
	cluster.LBSubsetKeys = sc.LBSubsetKeys
	// 关心两个环境的流量
	cluster.LBSubsetKeys = append(cluster.LBSubsetKeys, []string{sd.EnvKey}, []string{sd.EnvKey, ns.NAMESPACE})
	// 固化的默认策略
	cluster.LBDefaultKeys = []string{sd.EnvKey, "online"}
	cluster.Detector.DetectInterval = sc.DetectInterval
	cluster.Detector.ConsecutiveError = sc.ConsecutiveError
	cluster.Detector.ConsecutiveConnectionError = sc.ConsecutiveConnectionError
	cluster.Detector.MaxEjectionPercent = sc.MaxEjectionPercent
	cluster.Detector.SuccessRateMinHosts = sc.SuccessRateMinHosts
	cluster.Detector.SuccessRateRequestVolume = sc.SuccessRateRequestVolume
	cluster.Detector.SuccessRateStdevFactor = sc.SuccessRateStdevFactor
	cluster.Datacenter = sc.DC
	return cluster
}

func (d *Framework) initStat() {
	if len(d.LogDir) == 0 {
		utils.SetStat(filepath.Join(d.config.Log.LogPath, "stat"), d.localAppServiceName)
	} else {
		utils.SetStat(filepath.Join(d.LogDir, "stat"), d.localAppServiceName)
	}
	localSuccCodeMap := map[int]int{0: 1}
	for _, v := range d.config.Log.SuccStatCode {
		localSuccCodeMap[v] = 1
	}
	if !rfc.status() {
		metrics.AddSuccessCode(localSuccCodeMap)
		logging.GenLogf("on initStat, loading local success code:%+v", localSuccCodeMap)
	}
	// 监听变化并reload
	rfc.registerOnDisable(func() { // 关闭远程开关时,需要重新加载本地的状态码配置
		logging.GenLogf("on initStat, remote first disabled, reloading local success code:%+v", localSuccCodeMap)
		metrics.ReloadSuccessCode(localSuccCodeMap)
	})

	go func() {
		var lastValues []int
		type statCodeValues struct {
			Whitelist []int `json:"whitelist"`
		}
		remotePath := filepath.Join(getRegistryKVPath(d.localAppServiceName), "stat_code")
		w := config.WatchKV(remotePath)
		for {
			// blocked
			value := w.Next()
			if !rfc.status() {
				continue
			}
			v := value[remotePath]
			scv := &statCodeValues{}
			if err := reader.Encoder("json").Decode([]byte(v), scv); err != nil {
				continue
			}
			sort.Ints(scv.Whitelist)
			if reflect.DeepEqual(lastValues, scv.Whitelist) {
				continue
			}
			lastValues = scv.Whitelist
			newCodeMap := map[int]int{0: 1}
			for _, v := range lastValues {
				newCodeMap[v] = 1
			}
			metrics.ReloadSuccessCode(newCodeMap)
			logging.GenLogf("on initStat, config changed, reloading success code:%+v", newCodeMap)
		}
	}()
}

func (d *Framework) initGolangTrace() {
	if d.config.Trace.Disable {
		return
	}
	if d.config.Trace.Port == 0 {
		l, err := net.Listen("tcp", "0.0.0.0:0") // #nosec
		if err != nil {
			return
		}
		d.config.Trace.Port = l.Addr().(*net.TCPAddr).Port
		go http.Serve(l, nil)
	} else {
		go http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", d.config.Trace.Port), nil)
	}
	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) {
		return true, true
	}
}

func (d *Framework) initMiddleware() error {
	// init breaker and limiter
	d.initBreaker()
	d.initLimiter()

	kafkaProducerInit := func() error {
		return d.InitKafkaProducer(d.kafkaProducerConfig())
	}
	kafkaConsumerInit := func() error {
		return d.InitKafkaConsume(d.config.KafkaConsume)
	}
	redisClientInit := func() error {
		return d.InitRedisClient(d.redisConfig())
	}
	mysqlClientInit := func() error {
		return d.InitSqlClient(d.config.Database)
	}

	middlewares := map[string]func() error{
		"kafkaProducerInit": kafkaProducerInit,
		"kafkaConsumerInit": kafkaConsumerInit,
		"redisClientInit":   redisClientInit,
		"mysqlClientInit":   mysqlClientInit,
	}
	for name, fn := range middlewares {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		fnDone := make(chan error)
		go func() {
			fnDone <- fn()
		}()
	INNER:
		for {
			select {
			case <-ctx.Done():
				cancel()
				return fmt.Errorf("doing %s timeout, please check your config", name)
			case err := <-fnDone:
				if err != nil {
					cancel()
					return err
				}
				break INNER
			}
		}
		cancel()
	}

	return nil
}

func (d *Framework) kafkaProducerConfig() []kafka.KafkaProductConfig {
	var producerConfig []kafka.KafkaProductConfig
	if len(d.config.KafkaProducerClient) == 0 {
		return producerConfig
	}
	for _, defaultConfig := range d.config.KafkaProducerClient {
		var kpc kafka.KafkaProductConfig
		kpc.Broken = defaultConfig.Broken
		kpc.RetryMax = defaultConfig.RetryMax
		kpc.ProducerTo = defaultConfig.ProducerTo
		kpc.RequiredAcks = defaultConfig.Required_Acks
		kpc.GetError = defaultConfig.GetError
		kpc.GetSuccess = defaultConfig.GetSuccess
		kpc.UseSync = defaultConfig.Use_Sync
		producerConfig = append(producerConfig, kpc)
	}
	return producerConfig
}

func (d *Framework) redisConfig() []redis.RedisConfig {
	var redisConfig []redis.RedisConfig
	if len(d.config.Redis) == 0 {
		return redisConfig
	}
	for _, defaultConfig := range d.config.Redis {
		var rc redis.RedisConfig
		rc.ServerName = defaultConfig.ServerName
		rc.Addr = defaultConfig.Addr
		rc.Password = defaultConfig.Password
		rc.MaxIdle = defaultConfig.MaxIdle
		rc.MaxActive = defaultConfig.MaxActive
		rc.IdleTimeout = defaultConfig.IdleTimeout
		rc.ConnectTimeout = defaultConfig.ConnectTimeout
		rc.ReadTimeout = defaultConfig.ReadTimeout
		rc.WriteTimeout = defaultConfig.WriteTimeout
		rc.Database = defaultConfig.Database
		rc.Retry = defaultConfig.Retry
		redisConfig = append(redisConfig, rc)
	}
	return redisConfig
}
