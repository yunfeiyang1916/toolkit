package server

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/json"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/ecode"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/metric"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/namespace"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/tracing"
	dutils "github.com/yunfeiyang1916/toolkit/framework/utils"
	"github.com/yunfeiyang1916/toolkit/go-tls"
	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/logging"
	"golang.org/x/net/context"
)

type Server interface {
	Router
	Run(addr ...string) error
	Stop() error
	Use(p ...HandlerFunc)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type server struct {
	RouterMgr
	options        Options
	pluginMu       sync.Mutex
	plugins        []core.Plugin
	trees          methodTrees
	srv            *http.Server
	registryConfig *config.Register
	running        int32
	stop           chan struct{}
	once           sync.Once
	pool           sync.Pool
	paths          []string
	onHijackMode   bool
}

func NewServer(options ...Option) Server {
	s := &server{
		RouterMgr: RouterMgr{
			plugins:  nil,
			basePath: "/",
		},
		trees:    make(methodTrees, 0, 10),
		pluginMu: sync.Mutex{},
		plugins:  make([]core.Plugin, 0, 2),
		stop:     make(chan struct{}),
		pool:     sync.Pool{},
	}
	s.pool.New = func() interface{} {
		return s.allocContext()
	}
	s.options = newOptions(options...)
	s.srv = &http.Server{
		Handler:      s,
		ReadTimeout:  s.options.readTimeout,
		WriteTimeout: s.options.writeTimeout,
		IdleTimeout:  s.options.idleTimeout,
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateHijacked:
				s.onHijackMode = true
			}
		},
	}
	s.RouterMgr.server = s
	atomic.StoreInt32(&s.running, 0)

	return s
}

func (s *server) Run(addr ...string) error {
	var err error
	var host string
	s.once.Do(func() {
		s.uploadServerPath()
		port := 0
		if len(addr) > 0 {
			host = addr[0]
			tmp := strings.Split(host, ":")
			if len(tmp) == 2 {
				port, _ = strconv.Atoi(tmp[1])
			} else {
				err = fmt.Errorf("invalid addr: %s", addr)
				return
			}
		} else if s.options.port > 0 {
			port = s.options.port
			host = fmt.Sprintf(":%d", port)
		} else {
			host = ":80"
		}
		ln, e := net.Listen("tcp", host)
		if e != nil {
			logging.GenLogf("start http server on %s failed, %v", host, e)
			fmt.Printf("start http server on %s failed, %v\n", host, e)
			err = e
			return
		}
		logging.GenLogf("start http server on %s", host)
		fmt.Printf("start http server on %s\n", host)
		// 暂时注掉，暂不需要服务注册
		//var cfg *config.Register
		//cfg, err = dutils.Register(s.options.manager, s.options.serviceName, "http", s.options.tags, config.LocalIPString(), port)
		//if err != nil {
		//	return
		//}
		//s.registryConfig = cfg

		if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
			err = fmt.Errorf("server has been running")
			return
		}
		if len(s.options.certFile) == 0 || len(s.options.keyFile) == 0 {
			err = s.srv.Serve(ln)
		} else {
			err = s.srv.ServeTLS(ln, s.options.certFile, s.options.keyFile)
		}
		if err != nil {
			if err == http.ErrServerClosed {
				logging.GenLogf("http server closed: %v", err)
				err = nil
			}
		}
		logging.GenLogf("waiting for http server stop")
		fmt.Println("waiting for http server stop")
		// waiting for stop done
		<-s.stop
		logging.GenLogf("http server stop done")
		fmt.Println("http server stop done")
	})
	return err
}

func (s *server) Stop() error {
	if !atomic.CompareAndSwapInt32(&s.running, 1, 0) {
		return nil
	}

	defer close(s.stop)

	if s.options.manager != nil {
		s.options.manager.Deregister()
	}

	// gracefully shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := s.srv.Shutdown(ctx); err != nil {
		// Error from closing listeners, or context timeout:
		logging.GenLogf("gracefully shutdown, err:%v", err)
	}
	cancel()
	return nil
}

func (s *server) allocContext() *Context {
	return &Context{
		srv: s,
	}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		isSampled bool
		dumpReq   []byte
	)

	ctx := s.pool.Get().(*Context)
	ctx.reset()
	defer s.pool.Put(ctx)

	ctx.startTime = time.Now()
	ctx.w.reset(w, s.options.respBodyLogMaxSize)
	ctx.Request = r
	ctx.Response = ctx.w
	chain := ctx.chain()
	ctx.Ctx = tracing.HTTPToContext(s.options.tracer, r, fmt.Sprintf("HTTP Server %s %s", r.Method, ctx.Path))
	ctx.Namespace = namespace.GetNamespace(ctx.Ctx)
	ctx.Peer = metric.GetSDName(ctx.Ctx)

	span := opentracing.SpanFromContext(ctx.Ctx)
	defer span.Finish()

	ctx.extractBaggage()

	ext.PeerService.Set(span, ctx.Peer)
	ext.Component.Set(span, "inkelogic/go-http-server")
	span.LogFields(opentracinglog.String("event", "beginServe"))
	if !s.options.reqBodyLogOff && r.Body != nil {
		// piece reader
		lr := io.LimitReader(r.Body, 200)
		_, _ = ctx.bodyBuff.ReadFrom(lr)

		// rebuild body reader
		nr := bytes.NewBuffer(ctx.bodyBuff.Bytes())
		mr := io.MultiReader(nr, r.Body)
		ctx.Request.Body = ioutil.NopCloser(mr)
	}
	span.LogFields(opentracinglog.String("event", "read reqBody Done"))

	spanCtx := span.Context()
	if sc, ok := spanCtx.(jaeger.SpanContext); ok {
		ctx.traceId = sc.TraceID().String()
		// sampling record
		if sc.IsSampled() {
			isSampled = true
			dumpReq, _ = httputil.DumpRequest(r, true)
		}
	}
	curTime := ctx.startTime.Format(dutils.TimeFormat)
	if chain == nil {
		if s.methodNotAllowed(ctx) {
			span.LogFields(opentracinglog.String("event", "method not allowed"))
			ctx.Response.Header().Set("X-Trace-Id", ctx.traceId)
			ctx.Response.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = ctx.Response.Write([]byte(http.StatusText(http.StatusMethodNotAllowed)))
			fmt.Printf("%s http server, method not allowd, request %v\n", curTime, *r)
			return
		}
		span.LogFields(opentracinglog.String("event", "handlers not found"))
		ctx.Response.Header().Set("X-Trace-Id", ctx.traceId)
		ctx.Response.WriteHeader(http.StatusNotFound)
		_, _ = ctx.Response.Write([]byte(http.StatusText(http.StatusNotFound)))
		fmt.Printf("%s http server, handlers not found, request %v\n", curTime, *r)
		return
	}

	flow := core.New(chain)
	ctx.core = flow
	nCtx := context.WithValue(ctx.Ctx, iCtxKey, ctx)

	tls.SetContext(nCtx)
	defer tls.Flush()

	span.LogFields(opentracinglog.String("event", "start coreflow"))
	ctx.core.Next(nCtx)
	span.LogFields(opentracinglog.String("event", "coreflow Done"))

	// 优先使用用户设置的status, 服务执行出错设置status=500
	code := ctx.Response.Status()
	if !s.onHijackMode {
		if err := ctx.core.Err(); err != nil {
			ext.Error.Set(span, true)
			code = ecode.ConvertHttpStatus(err)
			ctx.Response.WriteHeader(code)
			_, _ = ctx.Response.WriteString(err.Error())
		}

		ctx.writeHeaderOnce()
	}

	span.SetTag("inkelogic.code", ctx.BusiCode())
	ext.HTTPStatusCode.Set(span, uint16(code))

	if isSampled {
		span.LogFields(
			opentracinglog.String("req", dutils.Base64(dumpReq)),
			opentracinglog.String("resp", dutils.Base64(ctx.Response.ByteBody())),
		)
	}
	span.LogFields(opentracinglog.String("event", "wrote Response Done"))
}

func (s *server) Use(p ...HandlerFunc) {
	s.pluginMu.Lock()
	defer s.pluginMu.Unlock()
	ps := make([]core.Plugin, len(p))
	for i := range p {
		ps[i] = p[i]
	}
	s.plugins = append(s.plugins, ps...)
}

func (s *server) addRoute(method, path string, handlers []core.Plugin) {
	if path[0] != '/' || len(method) == 0 || len(handlers) == 0 {
		return
	}
	root := s.trees.get(method)
	if root == nil {
		root = new(node)
		s.trees = append(s.trees, methodTree{method: method, root: root})
	}
	ps := s.makeChain(path, handlers)
	root.addRoute(path, ps)
	s.addPath(path)
}

func (s *server) addPath(path string) {
	var exist = false
	for _, v := range s.paths {
		if v == path {
			exist = true
			break
		}
	}
	if !exist {
		s.paths = append(s.paths, path)
	}
}

func (s *server) makeChain(path string, handlers []core.Plugin) []core.Plugin {
	// plugins list:
	// shared plugins on server: recover -> logging -> (maybe other plugin(s) inject on server)
	ps := []core.Plugin{
		s.traceIDHeader(),
		s.recover(),
		s.logging(),
	}
	s.pluginMu.Lock()
	for _, v := range s.plugins {
		vv := v
		ps = append(ps, vv)
	}
	s.pluginMu.Unlock()

	// third plugins effect on server, global scope
	gPlugins := serverInternalThirdPlugin.OnGlobalStage().Stream()
	ps = append(ps, gPlugins...)

	// plugins on each path: namespaceKey -> rateLimit-> breaker -> metric -> (outside frame plugin) -> handlers
	ps = append(ps, s.metric(), s.namespaceKey(), s.rateLimit(), s.breaker(path))

	// third plugins effect on server, global scope
	rPlugins := serverInternalThirdPlugin.OnRequestStage().Stream()
	ps = append(ps, rPlugins...)
	ps = append(ps, handlers...)
	return ps
}

func getRemoteIP(r *http.Request) string {
	for _, h := range []string{"X-Real-Ip"} {
		addresses := strings.Split(r.Header.Get(h), ",")
		for i := len(addresses) - 1; i >= 0; i-- {
			ip := addresses[i]
			if len(ip) > 0 {
				return ip
			}
		}
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func (s *server) uploadServerPath() {
	body := map[string]interface{}{}
	body["type"] = 1
	body["resource_list"] = s.paths
	body["service"] = s.options.serviceName
	b, _ := json.NewEncoder().Encode(body)
	respB, err := tracing.KVPut(b)
	if err != nil {
		return
	}
	logging.GenLogf("sync http server path list to consul response:%q", respB)
}
