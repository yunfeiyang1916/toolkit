package server

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	openlog "github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/metric"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/namespace"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/tracing"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/internal/ikiosocket"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/internal/metadata"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/internal/rpcerror"
	"github.com/yunfeiyang1916/toolkit/framework/utils"
	"github.com/yunfeiyang1916/toolkit/go-tls"
	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/metrics"
	"golang.org/x/net/context"
)

type binaryServer struct {
	opts     Options
	router   *router
	server   *ikiosocket.Server
	plugins  []Plugin
	cfg      *config.Register
	shutdown int32
	stop     chan struct{}
	once     sync.Once
}

func BinaryServer(options ...Option) Server {
	opts := newOptions(options...)
	h := &binaryServer{}
	h.router = newRouter()
	h.opts = opts
	h.server = ikiosocket.NewServer(logging.Log(logging.GenLoggerName), h.serveBinary)
	h.shutdown = 0
	h.stop = make(chan struct{})
	return h
}

func (h *binaryServer) serveBinary(remoteAddr string, request *ikiosocket.Context) (*ikiosocket.Context, error) {
	var (
		codec    = h.opts.Codec
		start    = time.Now()
		traceId  string
		errStr   string
		rpcApi   = "unknown"
		retCode  = 0
		nullBody = []byte{}
	)
	ctx := tracing.BinaryToContext(h.opts.Tracer, request.Header, "RPC Server", nil)
	tls.SetContext(ctx)
	ns := namespace.GetNamespace(ctx)
	peer := metric.GetSDName(ctx)
	span := opentracing.SpanFromContext(ctx)
	span.SetTag("proto", "rpc/rpcBin")
	spanCtx := span.Context()
	if sc, ok := spanCtx.(jaeger.SpanContext); ok {
		traceId = sc.TraceID().String()
	}

	ext.PeerService.Set(span, peer)

	defer span.Finish()
	defer tls.Flush()
	defer func() {
		if h.opts.Kit.A() == nil {
			return
		}
		logItems := []interface{}{
			"start", start.Format(utils.TimeFormat),
			"cost", math.Ceil(float64(time.Since(start).Nanoseconds()) / 1e6),
			"trace_id", traceId,
			"peer_name", peer,
			"rpc_method", rpcApi,
			"rpc_code", retCode,
			"namespace", ns,
			"real_ip", remoteAddr,
			"err", errStr,
		}
		h.opts.Kit.A().Debugw("rpcserver-rpc", logItems...)
	}()

	handleResponse := func(code int, id uint64, desc, name string, header map[string]string, bodyByte []byte) *ikiosocket.Context {
		if ns != "" {
			metrics.Timer(name, start, metrics.TagCode, code, "namespace", ns)
		} else {
			metrics.Timer(name, start, metrics.TagCode, code)
		}
		retCode = code
		failed := code != 0
		if failed {
			ext.Error.Set(span, true)
		}
		meta := &metadata.RpcMeta{
			Type:       metadata.RpcMeta_RESPONSE.Enum(),
			SequenceId: proto.Uint64(id),
			Failed:     proto.Bool(failed),
			ErrorCode:  proto.Int32(int32(code)),
			Reason:     proto.String(desc),
		}
		metaByte, _ := proto.Marshal(meta)
		if header == nil {
			header = map[string]string{}
		}
		header[metadata.MetaHeaderKey] = string(metaByte)
		header[metadata.DataHeaderKey] = string(bodyByte)
		return &ikiosocket.Context{
			Header: header,
		}
	}

	if _, ok := request.Header[metadata.MetaHeaderKey]; !ok {
		errStr = "Meta Key Not Exist"
		return handleResponse(rpcerror.Internal, 0, errStr, "SServer", nil, nullBody), nil
	}
	if _, ok := request.Header[metadata.DataHeaderKey]; !ok {
		errStr = "Data Key Not Exist"
		return handleResponse(rpcerror.Internal, 0, errStr, "SServer", nil, nullBody), nil
	}

	span.LogFields(openlog.String("event", "decode RPC Meta"))
	meta := metadata.RpcMeta{}
	if err := proto.Unmarshal([]byte(request.Header[metadata.MetaHeaderKey]), &meta); err != nil {
		errStr = "server decode rpc Meta failed," + err.Error()
		span.LogFields(openlog.Error(err))
		return handleResponse(rpcerror.Internal, 0, errStr, "SServer", nil, nullBody), nil
	}
	rpcApi = meta.GetMethod()
	pos := strings.LastIndex(rpcApi, ".")
	if pos == -1 {
		errStr = "invalid request, undefined/malformed method name"
		return handleResponse(rpcerror.Internal, meta.GetSequenceId(), errStr, "SServer", nil, nullBody), nil
	}
	service := rpcApi[:pos]
	method := rpcApi[pos+1:]
	metricName := fmt.Sprintf("SServer.%s.%s", service, method)
	span.SetOperationName(fmt.Sprintf("RPC Server %s.%s", service, method))
	stype, mtype, args, err := h.router.signature(service, method)
	if err != nil {
		errStr = fmt.Sprintf("invalied request, api unknown %s,%v", rpcApi, err)
		span.LogFields(openlog.Error(err))
		return handleResponse(rpcerror.Internal, meta.GetSequenceId(), errStr, metricName, nil, nullBody), nil
	}
	body := []byte(request.Header[metadata.DataHeaderKey])

	c := core.New(nil)
	rpcctx := &Context{
		core:       c,
		opts:       h.opts,
		Ctx:        ctx,
		Service:    service,
		Method:     method,
		Header:     make(map[string]string),
		RemoteAddr: remoteAddr,
		Body:       body,
		Request:    args,
		Code:       int32(rpcerror.Success),
		Namespace:  ns,
		Peer:       peer,
	}

	span.LogFields(openlog.String("event", "decode request body"))
	if err := codec.Decode(body, rpcctx.Request); err != nil {
		errStr = "server parse request failed," + err.Error()
		span.LogFields(openlog.Error(err))
		return handleResponse(rpcerror.Internal, meta.GetSequenceId(), errStr, metricName, nil, nullBody), nil
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

	span.LogFields(openlog.String("event", "rpc serving"))
	rpcctx.Next()
	if err := rpcctx.Err(); err != nil {
		errStr = "server handling request failed," + err.Error()
		span.LogFields(openlog.Error(err))
		return handleResponse(rpcerror.FromUser, meta.GetSequenceId(), errStr, metricName, nil, nullBody), nil
	}

	span.LogFields(openlog.String("event", "encode response"))
	body, err = codec.Encode(rpcctx.Response)
	if err != nil {
		errStr = "server encode response failed," + err.Error()
		span.LogFields(openlog.Error(err))
		return handleResponse(rpcerror.Internal, meta.GetSequenceId(), errStr, metricName, nil, nullBody), nil
	}

	// reload code,maybe changed by outside logic
	atomic.LoadInt32(&rpcctx.Code)
	return handleResponse(int(rpcctx.Code), meta.GetSequenceId(), "", metricName, rpcctx.Header, body), nil
}

func (h *binaryServer) NewHandler(handler interface{}, opts ...HandlerOption) Handler {
	return h.router.NewHandler(handler, opts...)
}

func (h *binaryServer) Handle(handler Handler) error {
	return h.router.Handle(handler)
}

func (h *binaryServer) Use(list ...Plugin) Server {
	h.plugins = append(h.plugins, list...)
	return h
}

func (h *binaryServer) Start() error {
	var err error
	h.once.Do(func() {
		ln, e := net.Listen("tcp4", h.opts.Address)
		if e != nil {
			logging.GenLogf("start rpc server on %s failed, %v", h.opts.Address, e)
			fmt.Printf("start rpc server on %s failed, %v\n", h.opts.Address, e)
			err = e
			return
		}

		addrStr := ln.Addr().String()
		logging.GenLogf("start rpc server on %s", addrStr)
		fmt.Printf("start rpc server on %s\n", addrStr)
		addr := strings.Split(addrStr, ":")
		port, _ := strconv.Atoi(addr[1])
		cfg, e := utils.Register(
			h.opts.Manager, h.opts.Name, "rpc", h.opts.Tags, config.LocalIPString(), port)
		if e != nil {
			err = e
			return
		}

		h.cfg = cfg
		err = h.server.Start(ln)
		if strings.Contains(err.Error(), "use of closed network connection") {
			err = nil
		}
		logging.GenLogf("waiting for rpc server stop")
		fmt.Println("waiting for rpc server stop")
		// waiting for stop done
		<-h.stop
		logging.GenLogf("rpc server stop done")
		fmt.Println("rpc server stop done")
	})
	return err
}

func (h *binaryServer) Stop() error {
	if !atomic.CompareAndSwapInt32(&h.shutdown, 0, 1) {
		return nil
	}

	defer close(h.stop)

	if m := h.opts.Manager; m != nil {
		m.Deregister()
	}
	h.server.Stop()
	return nil
}

func (h *binaryServer) GetPaths() []string {
	return h.router.GetPaths()
}
