package gin

import (
	"net/http"
	"sync"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// HandlerFunc 路由及中间件处理程序
type HandlerFunc func(ctx *Context)

// HandlersChain 处理程序链
type HandlersChain []HandlerFunc

type Engine struct {
	// 路由组
	RouterGroup
	// 负责存储路由和handle方法的映射,采用类似字典树的结构
	trees methodTrees
	pool  sync.Pool
	// UseH2C enable h2c support.
	UseH2C bool
}

// Run 启动Http监听服务，该方法会阻塞调用协程
func (engine *Engine) Run(addr ...string) (err error) {
	address := resolveAddress(addr)
	debugPrint("Listening and serving HTTP on %s\n", address)
	err = http.ListenAndServe(address, engine.Handler())
	return
}

func (engine *Engine) Handler() http.Handler {
	if !engine.UseH2C {
		return engine
	}

	h2s := &http2.Server{}
	return h2c.NewHandler(engine, h2s)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := engine.pool.Get().(*Context)
	c.Request = req
	c.reset()

	engine.handleHTTPRequest(c)
	engine.pool.Put(c)
}

func (engine *Engine) handleHTTPRequest(c *Context) {
	httpMethod := c.Request.Method
	rPath := c.Request.URL.Path

	// 获取压缩前缀树数组，每个请求方法都有一颗radix树
	t := engine.trees
	for i, tl := 0, len(t); i < tl; i++ {
		// 找到当前请求方式对应的radix树,每个http method都有一颗前缀树
		if t[i].method != httpMethod {
			continue
		}
		// 得到树的根节点
		root := t[i].root
		// 根据请求路径获取匹配的redix树节点
		value := root.getValue(rPath, c.params, c.skippedNodes, unescape)
		if value.params != nil {
			c.Params = *value.params
		}
		if value.handlers != nil {
			c.handlers = value.handlers
			c.fullPath = value.fullPath
			c.Next()
			c.writermem.WriteHeaderNow()
			return
		}
		if httpMethod != http.MethodConnect && rPath != "/" {
			if value.tsr && engine.RedirectTrailingSlash {
				redirectTrailingSlash(c)
				return
			}
			if engine.RedirectFixedPath && redirectFixedPath(c, root, engine.RedirectFixedPath) {
				return
			}
		}
		break
	}
}
