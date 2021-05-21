package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"

	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/retry"
	"golang.org/x/net/context"
)

// client interface
type Client interface {
	Call(*Request) (*Response, error)
	Use(p ...HandlerFunc)
}

type Func func(*Request) (*Response, error)

func (f Func) Call(req *Request) (*Response, error) {
	return f(req)
}

func (f Func) Use(p ...HandlerFunc) {
}

type client struct {
	client  *http.Client
	options Options
	mu      sync.Mutex
	ps      []core.Plugin
	static  []core.Plugin
}

func NewClient(opts ...Option) Client {
	c := &client{}
	c.options = newOptions(opts...)
	if c.options.client != nil {
		c.client = c.options.client
	} else {
		tp := &http.Transport{
			// 表示对每台host保持ESTABLISHED连接状态的最大链接数量,默认100
			MaxIdleConnsPerHost: c.options.maxIdleConnsPerHost,

			// 表示对所有host保持ESTABLISHED连接状态的最大链接数量,默认100
			MaxIdleConns: c.options.maxIdleConns,

			Proxy: http.ProxyFromEnvironment,

			// 该函数用于创建http（非https）连接
			DialContext: (&net.Dialer{
				// 表示建立Tcp链接超时时间,默认10s
				Timeout: c.options.dialTimeout,

				// 表示底层为了维持http keepalive状态,每隔多长时间发送Keep-Alive报文
				// 通常要与IdleConnTimeout对应,默认30s
				KeepAlive: c.options.keepAliveTimeout,
			}).DialContext,

			// 连接最大空闲时间,超过这个时间就会被关闭,也即socket在该时间内没有交互则自动关闭连接
			// 该timeout起点是从每次空闲开始计时,若有交互则重置为0,该参数通常设置为分钟级别,默认90s
			IdleConnTimeout: c.options.idleConnTimeout,

			// 限制TLS握手使用的时间
			TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
			ExpectContinueTimeout: defaultExpectContinueTimeout,

			// 表示是否开启http keepalive功能，也即是否重用连接，默认开启(false)
			DisableKeepAlives: c.options.keepAlivesDisable,
		}

		if httpsInsecureSkipVerify {
			tp.TLSClientConfig = &tls.Config{InsecureSkipVerify: httpsInsecureSkipVerify} // #nosec
		}

		c.client = &http.Client{
			Transport: tp,
			// 此client的请求处理时间,包括建连,重定向,读resp所需的所有时间
			// Timeout: c.options.requestTimeout,
		}
	}
	c.static = c.chain()
	return c
}

func (c *client) chain() []core.Plugin {
	ps := []core.Plugin{
		c.recover(),
		c.retry(),
		c.tracing(),
		c.namespace(),
		c.peername(),
		c.logging(),
	}
	// host discovery
	if u := c.upstream(); u != nil {
		ps = append(ps, u)
	}

	// maybe modify req by outside plugin
	gPlugins := clientInternalThirdPlugin.OnGlobalStage().Stream()
	ps = append(ps, gPlugins...)
	ps = append(ps, c.delayPlugins())

	// outside plugins
	rPlugins := clientInternalThirdPlugin.OnRequestStage().Stream()
	ps = append(ps, rPlugins...)

	// fixed build-in plugins
	ps = append(ps, c.buildRequest(), c.rateLimit(), c.breaker(), c.sender())
	return ps
}

func (c *client) Call(r *Request) (*Response, error) {
	// use config value by default
	if len(r.scheme) == 0 {
		r.scheme = c.options.protoType
	}
	if r.ro.reqTimeout == 0 {
		r.ro.reqTimeout = c.options.requestTimeout
	}
	if r.ro.slowTime == 0 {
		r.ro.slowTime = c.options.slowTimeout
	}
	if r.ro.retryTimes == 0 {
		r.ro.retryTimes = c.options.retryTimes
	}

	ctx := newContext(r)
	ctx.core = core.New(c.static)

	nCtx := context.WithValue(ctx.Ctx, iCtxKey, ctx)
	nCtx = context.WithValue(nCtx, retry.Key, r.ro.retryTimes)

	ctx.core.Next(nCtx)
	return ctx.Resp, ctx.core.Err()
}

func (c *client) Use(p ...HandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ps == nil {
		c.ps = make([]core.Plugin, len(p))
		for i, v := range p {
			c.ps[i] = v
		}
	} else {
		for _, v := range p {
			c.ps = append(c.ps, v)
		}
	}
}
