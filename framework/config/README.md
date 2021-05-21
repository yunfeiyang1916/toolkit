# Daenerys Config模块使用手册

> Daenerys Config模块主要用于解析配置文件，支持多数据源加载，多文件加载，动态监听等功能。
> 目前最常用的使用方式是加载本地配置文件和远程配置文件。

## 加载配置文件

### 1.本地配置文件
> 一般使用Daenerys框架时,默认情况下会使用约定路径下的配置文件,即服务运行路径下的 config/config.toml。
手动启动服务命令示例：./app 或者 ./app --config ./config/config.toml

框架加载配置文件的代码示例，以下代码是通过代码模版工具生成的，无需使用者手动添加。
```go
func init() {
	configS := flag.String("config", "config/config.toml", "Configuration file")
	flag.Parse()
	// 加载配置文件
	daenerys.Init(daenerys.ConfigPath(*configS))
}
```

如果需要手动加载其他配置文件内容，可以通过以下附加方式实现。加载后会将之前的配置和本次加载的配置合并。
* 注意：如果两个文件中存在相同配置项，那么后加载的会覆盖前者。使用者应该避免这种情况。
```go
err := daenerys.File("/abs/path/xxx.json")
if err!=nil{
	panic(err)
}
```

### 2.加载远程配置文件
> 框架在加载本地配置文件的同时也会默认加载远程路径下的配置。
假设某服务的服务发现名是aa.bb.cc，那么默认的远程路径是：
/service_config/aa/aa.bb.cc/config.toml
* 注意：由于远程方式是后加载的，当本地配置与远程配置有相同配置项时，那么将以远程配置为准。
如果需要手动加载远程配置，可以通过以下附加方式实现。加载后会将之前的配置和本次加载的配置合并。
```go
// 短路径
err := daenerys.Remote("/xxx")
if err != nil {
	panic(err)
}
```


### 3.加载namespace配置文件
> 为了支持同一个服务可以根据不同的配置内容来调用不同的资源，框架支持了namespace配置加载方式。
> 同样，框架会自动加载配置，不同的是，需要加上额外的参数，以下代码也是通过代码模版工具生成。无需手动添加。
```go
	configS := flag.String("config", "config/config.toml", "Configuration file")
	appS := flag.String("app", "", "App dir")
	flag.Parse()
	daenerys.Init(
		daenerys.ConfigPath(*configS),
	)
	if *appS != "" {
		daenerys.InitNamespace(*appS)
	}
```
使用多namespace方式启动服务：
```go
./app --app ./config/app
```
使用namespace时，配置文件的目录结构如下：
```text
├── app
│   ├── meetstar
│   │   └── config.toml
│   └── starstar
│       └── config.toml
└── config.toml
```
* 注意：app目录下的子目录meetstar和starstar将会任务是namespace名，用于区分不同的配置域，不同的配置资源。

> 同样，在使用namespace时，也会自动加载一个默认的远程路径配置。
> 假设上面示例的服务名是aa.bb.cc, 那么默认远程路径为：/service_config/aa/aa.bb.cc/meetstar/config.toml 和 /service_config/aa/aa.bb.cc/starstar/config.toml
> 同样远程配置内容会优先生效。


## 获取配置文件内容
> 在配置文件加载成功后，可以通过以下方式获取一个Config实例，通过它我们可以做一些更为丰富的操作。
> Config实例提供的接口如下：
```go

type Config interface {
	reader.Values
	LoadFile(f ...string) error
	LoadPath(p string, isPrefix bool, format string) error
	Sync() error
	Listen(interface{}) loader.Refresher
}

type Values interface {
	Bytes() []byte
	String() string
	Get(keys ...string) Value
	Map() map[string]interface{}
	Range(f func(k string, v interface{}))
	Scan(v interface{}) error
}
```

### 1.通用使用方式

>下面介绍一些通用的使用方式：
```go
    // 获取之前加载配置文件后的Config实例
    c := daenerys.ConfigInstance()

    // 重新加载所有资源，获取最新配置
    err := c.Sync()
    if err != nil{
	    panic(err)
    }

    // 将配置内容转为byte
    c.Bytes()

    // 将配置内容转为string
    c.String()

    // 将配置内容转为map
    c.Map()

    // 以map方式遍历配置
    c.Range(func(k string, v interface{}) {
	    fmt.Println("key:", k, "value:",v)
    })

    // 获取某个配置项的值
    v := c.Get("a","b", "c")
    vBool := v.Bool(false) // 取bool值,默认false
    vInt := v.Int(0) // 取整型值,默认0
    vStr := v.String("unknown") // 取string值,默认unknown

    // 将配置内容解析到一个自定义的struct中, v需要是个指针
    type xxx struct {
	
    }
    v := &xxx{}
    err := c.Scan(v)
    if err != nil {
	    // todo sth
    }

    // 动态监听一个struct变动, v需要是个指针
    l := c.Listen(v)
    // todo sth

    newv := r.Load().(*xxx)

```

### 2.namespace使用方式
> 由于namespace方式在框架加载配置时候就将配置文件资源做了隔离，所以使用方式与通用的稍微有点不同。
> 首先需要通过ctx方式获取相应的Config实例，之后的使用方式与通用的就一样了。
> ctx中需要设置上相应的namespace值，ConfigInstanceCtx会根据该值获取对应的Config实例。

以下是示例代码，此处只是演示使用，通常情况下无需手动设置appkey，appkey会从请求中自动提取到。
```go
	ctx := daenerys.WithAPPKey(context.Background(),"meetstar")
    // 获取Config实例,使用方式见：通用使用方式
    c := daenerys.ConfigInstanceCtx(ctx)
```

### 监听指定path的远程配置
> 框架默认情况下会将加载进去的资源做数据合并处理，当其中某一个资源变动时，所有配置内容就会被重新加载和整合。
> 有些情况下，我们只关心某个远程路径下的资源是否有变动。所以我们提供了比较便捷的方式来监听这种更新。

以下是示例代码：
```go

    // 监听远程绝对路径：/service_config/aa/aa.bb.cc/test，当test中内容有更新时返回数据
	go func() {
		p := "/test"
		w := daenerys.WatchKV(p)
		for {
			// will block
			value := w.Next()
			fmt.Println("watch kv>>>>>>>>", value[p])
		}
	}()

    // 远程资源：
    // 路径1：/service_config/aa/aa.bb.cc/test/a/a.toml
    // 路径2：/service_config/aa/aa.bb.cc/test/b/b.toml
    
    // 监听远程路径前缀：/service_config/aa/aa.bb.cc/test, 当a.toml或b.toml有更新时返回数据
    // Next返回值是个map, key为短路径：/a/a.toml, /b/b.toml, value为该路径对应的值
	go func() {
		p := "/test"
		w := daenerys.WatchPrefix(p)
		for {
			// will block
			value := w.Next()
			fmt.Println("watch prefix>>>>>>>>", value)
		}
	}()

    // 直接获取远程配置
    str, err := daenerys.RemoteKV(p)
    if err != nil {
    	panic(err)
    }
    

```

















