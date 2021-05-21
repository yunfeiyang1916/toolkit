package server

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
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
	openlog "github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/metric"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/namespace"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/tracing"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/internal/rpcerror"
	"github.com/yunfeiyang1916/toolkit/framework/utils"
	"github.com/yunfeiyang1916/toolkit/go-tls"
	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/metrics"
	"golang.org/x/net/context"
)

type httpServer struct {
	opts     Options
	router   *router
	srv      *http.Server
	plugins  []Plugin
	cfg      *config.Register
	stop     chan struct{}
	shutdown int32
	once     sync.Once
}

func HTTPServer(options ...Option) Server {
	opts := newOptions(options...)
	h := &httpServer{}
	h.router = newRouter()
	h.opts = opts
	h.srv = &http.Server{
		Addr:      opts.Address,
		Handler:   h,
		ConnState: nil, // TODO
	}
	h.stop = make(chan struct{})
	h.shutdown = 0
	return h
}

func (h *httpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		isSampled  bool
		dumpReq    []byte
		codec      = h.opts.Codec
		remoteAddr = r.RemoteAddr
		start      = time.Now()
		traceId    string
		rpcApi     string
		metricName string
		retCode    = 0
		errStr     string
	)
	optName := fmt.Sprintf("RPC Server %s %s", r.Method, r.URL.Path)
	ctx := tracing.HTTPToContext(h.opts.Tracer, r, optName)
	tls.SetContext(ctx)
	peer := metric.GetSDName(ctx)
	ns := namespace.GetNamespace(ctx)
	span := opentracing.SpanFromContext(ctx)
	spanCtx := span.Context()
	// record HttpRPC request
	if sc, ok := spanCtx.(jaeger.SpanContext); ok {
		// sampling record
		if sc.IsSampled() {
			isSampled = true
			dumpReq, _ = httputil.DumpRequest(r, true)
		}
		traceId = sc.TraceID().String()
	}

	span.SetTag("proto", "rpc/http")
	ext.PeerService.Set(span, peer)

	defer tls.Flush()
	defer span.Finish()
	defer func() {
		if h.opts.Kit.A() == nil {
			return
		}
		logItems := []interface{}{
			"start", start.Format(utils.TimeFormat),
			"cost", math.Ceil(float64(time.Since(start).Nanoseconds()) / 1e6),
			"trace_id", traceId,
			"peer_name", peer,
			"req_uri", r.URL.Path,
			"rpc_method", rpcApi,
			"rpc_code", retCode,
			"namespace", ns,
			"real_ip", remoteAddr,
			"err", errStr,
		}
		h.opts.Kit.A().Debugw("rpcserver-http", logItems...)
	}()

	handleError := func(code int, dest string, name string) {
		retCode = code
		if code != 0 {
			ext.Error.Set(span, true)
			http.Error(w, rpcerror.HTTPError{
				C:    code,
				Desc: dest,
			}.Marshal(), http.StatusBadRequest)
		}
		if ns != "" {
			metrics.Timer(name, start, metrics.TagCode, code, "namespace", ns)
		} else {
			metrics.Timer(name, start, metrics.TagCode, code)
		}
	}

	rpcApi = strings.Replace(r.URL.Path[1:], "/", ".", -1)
	pos := strings.LastIndex(rpcApi, ".")
	if pos == -1 {
		errStr = "invalid request, undefined/malformed method name"
		handleError(rpcerror.Internal, errStr, "HServer")
		span.LogFields(
			openlog.String("event", "decode meta data"),
			openlog.Error(errors.New(errStr)))
		return
	}

	service := rpcApi[:pos]
	method := rpcApi[pos+1:]
	metricName = fmt.Sprintf("HServer.%s.%s", service, method)
	span.SetOperationName(fmt.Sprintf("RPC Server %s.%s", service, method))

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errStr = "server read body failed," + err.Error()
		handleError(rpcerror.Internal, errStr, metricName)
		span.LogFields(
			openlog.String("event", "read body"),
			openlog.Error(err),
		)
		return
	}

	stype, mtype, args, err := h.router.signature(service, method)
	if err != nil {
		errStr = fmt.Sprintf("invalied request, api unknown %s,%v", rpcApi, err)
		handleError(rpcerror.Internal, errStr, metricName)
		span.LogFields(openlog.Error(err))
		return
	}

	c := core.New(nil)
	rpcctx := &Context{
		core:       c,
		opts:       h.opts,
		Ctx:        ctx,
		Service:    service,
		Method:     method,
		Peer:       peer,
		Header:     make(map[string]string),
		RemoteAddr: remoteAddr,
		Body:       body,
		Request:    args,
		Code:       int32(rpcerror.Success),
		Namespace:  ns,
	}

	span.LogFields(openlog.String("event", "decode request body"))
	if len(body) == 0 {
		body = []byte("{}")
	}
	if err := codec.Decode(body, rpcctx.Request); err != nil {
		errStr = "server parse request failed," + err.Error()
		handleError(rpcerror.Internal, errStr, metricName)
		span.LogFields(openlog.Error(err))
		return
	}

	for _, plugin := range h.plugins {
		p := plugin
		c.Use(core.Function(func(ctx context.Context, c core.Core) {
			p(rpcctx)
		}))
	}

	c.Use(core.Function(func(ctx context.Context, c core.Core) {
		response, err := h.router.call(rpcctx.Ctx, stype, mtype, rpcctx.Request)
		if err != nil {
			c.AbortErr(err)
			return
		}
		rpcctx.Response = response
	}))

	span.LogFields(openlog.String("event", "http serving"))
	rpcctx.Next()

	if err := rpcctx.Err(); err != nil {
		errStr = "server handling request failed," + err.Error()
		handleError(rpcerror.FromUser, errStr, metricName)
		span.LogFields(openlog.Error(err))
		return
	}

	span.LogFields(openlog.String("event", "encode response"))
	body, err = codec.Encode(rpcctx.Response)
	if err != nil {
		errStr = "server encode response failed," + err.Error()
		handleError(rpcerror.Internal, errStr, metricName)
		span.LogFields(openlog.Error(err))
		return
	}

	span.LogFields(openlog.String("event", "write response"))
	if _, err := w.Write(body); err != nil {
		errStr = "server write response failed," + err.Error()
		handleError(rpcerror.Internal, errStr, metricName)
		span.LogFields(openlog.Error(err))
		return
	}
	atomic.LoadInt32(&rpcctx.Code)
	handleError(int(rpcctx.Code), "", metricName)

	// record HttpRPC req&resp
	if isSampled {
		span.LogFields(
			openlog.String("req", utils.Base64(dumpReq)),
			openlog.String("resp", utils.Base64(body)),
		)
	}
}

func (h *httpServer) NewHandler(handler interface{}, opts ...HandlerOption) Handler {
	return h.router.NewHandler(handler, opts...)
}

func (h *httpServer) Handle(handler Handler) error {
	return h.router.Handle(handler)
}

func (h *httpServer) Use(list ...Plugin) Server {
	h.plugins = append(h.plugins, list...)
	return h
}

func (h *httpServer) Start() error {
	var err error
	h.once.Do(func() {
		ln, e := net.Listen("tcp4", h.opts.Address)
		if e != nil {
			logging.GenLogf("start rpc-http server on %s failed, %v", h.opts.Address, e)
			fmt.Printf("start rpc-http server on %s failed, %v\n", h.opts.Address, e)
			err = e
			return
		}

		addrStr := ln.Addr().String()
		logging.GenLogf("start rpc-http server on %s", addrStr)
		fmt.Printf("start rpc-http server on %s\n", addrStr)
		addr := strings.Split(addrStr, ":")
		port, _ := strconv.Atoi(addr[1])
		cfg, e := utils.Register(
			h.opts.Manager, h.opts.Name, "http", h.opts.Tags, config.LocalIPString(), port)
		if e != nil {
			err = e
			return
		}
		h.cfg = cfg

		err = h.srv.Serve(ln)
		if err != nil {
			if err == http.ErrServerClosed {
				logging.GenLogf("rpc-http server closed: %v", err)
				err = nil
			}
		}
		logging.GenLogf("waiting for rpc-http server stop")
		fmt.Println("waiting for rpc-http server stop")
		// waiting for stop done
		<-h.stop
		logging.GenLogf("rpc-http server stop done")
		fmt.Println("rpc-http server stop done")
	})
	return err
}

func (h *httpServer) Stop() error {
	if !atomic.CompareAndSwapInt32(&h.shutdown, 0, 1) {
		return nil
	}

	defer close(h.stop)

	if m := h.opts.Manager; m != nil {
		m.Deregister()
	}

	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := h.srv.Shutdown(ctx); err != nil {
		logging.GenLogf("gracefully shutdown, err:%v", err)
	}
	cancel()
	return err
}

func (h *httpServer) GetPaths() []string {
	return h.router.GetPaths()
}
