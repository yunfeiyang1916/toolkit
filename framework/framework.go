package framework

import (
	"sync"

	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
	"github.com/yunfeiyang1916/toolkit/go-upstream/upstream"

	"honnef.co/go/tools/config"
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
