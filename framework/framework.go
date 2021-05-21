package framework

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yunfeiyang1916/toolkit/framework/config"
	"github.com/yunfeiyang1916/toolkit/framework/utils"
	"github.com/yunfeiyang1916/toolkit/logging"

	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
	"github.com/yunfeiyang1916/toolkit/go-upstream/upstream"
)

type Framework struct {
	// Name is discovery name, it is from deploy platform by default.
	// Name will be used to register to discovery service
	Name      string
	Namespace string
	JobName   string
	App       string
	// 版本
	Version                     string
	Deps                        string
	LogDir                      string
	LogLevel                    string
	LogRotate                   string
	ConfigPath                  string
	ConfigMemory                []byte
	Clusters                    *upstream.ClusterManager
	Manager                     *registry.ServiceManager
	config                      frameworkConfig
	configInstance              config.Config
	redisClients                sync.Map
	mysqlClients                sync.Map
	consumeClients              sync.Map
	producerClients             sync.Map
	serverClientMap             sync.Map
	httpClientMap               sync.Map
	namespaceConfig             sync.Map
	rpcClientMap                sync.Map
	mu                          sync.Mutex
	localAppServiceName         string
	pendingServerClientTask     []ServerClient
	pendingServerClientLock     sync.Mutex
	pendingServerClientTaskDone int32
	initOnce                    sync.Once
	namespaceDir                string
}

func New() *Framework {
	return &Framework{
		Name:                        "",
		Namespace:                   "",
		App:                         "",
		Version:                     "",
		LogDir:                      "logs",
		LogLevel:                    "debug",
		LogRotate:                   "hour",
		ConfigPath:                  "",
		Clusters:                    upstream.NewClusterManager(),
		Manager:                     nil,
		configInstance:              nil,
		pendingServerClientTask:     nil,
		pendingServerClientLock:     sync.Mutex{},
		pendingServerClientTaskDone: 0,
		initOnce:                    sync.Once{},
	}
}

func (d *Framework) Init(options ...Option) {
	d.initOnce.Do(func() {
		for _, opt := range options {
			opt(d)
		}
		if len(d.JobName) == 0 {
			d.JobName = strings.TrimSpace(readFile(".jobname"))
		}
		if len(d.Deps) == 0 {
			d.Deps = readFile(".deps")
		}
		if len(d.App) == 0 {
			d.App = strings.TrimSpace(readFile(".app"))
		}
		if len(d.Version) == 0 {
			d.Version = strings.TrimSpace(readFile(".version"))
		}

		// 读取本地配置
		var cc []byte
		if len(d.ConfigMemory) > 0 {
			cc = d.ConfigMemory
		} else {
			cc = d.loadLocalConfig()
		}

		// 设置服务发现名
		d.initLocalAppServiceName(cc)

		// 处理远程开关逻辑
		initRemoteFirst(d.localAppServiceName)

		d.pendingServerClientLock.Lock()
		pending := d.pendingServerClientTask
		d.pendingServerClientLock.Unlock()

		if len(cc) > 0 {
			if err := d.initConfigInstance(); err != nil {
				panic(err)
			}

			// logger,consul backend,tracer都只初始化一次
			if len(d.ConfigPath) > 0 {
				d.initDefaultOnce()
			}

			d.Manager = registry.NewServiceManager(logging.Log(logging.BalanceLoggerName))

			// init middleware client
			if err := d.initMiddleware(); err != nil {
				panic(err)
			}

			// inject service client from config
			d.pendingServerClientLock.Lock()
			pending = append(pending, d.config.ServerClient...)
			d.pendingServerClientTask = nil
			atomic.StoreInt32(&d.pendingServerClientTaskDone, 1)
			d.pendingServerClientLock.Unlock()
		}

		for _, sc := range pending {
			d.injectServerClient(sc)
		}

		curTime := time.Now().Format(utils.TimeFormat)
		fmt.Printf("%s init framework success app:%s name:%s namespace:%s config:%s\n",
			curTime, d.App, d.Name, d.Namespace, d.ConfigPath)
	})
}
