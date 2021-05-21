package client

import (
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	openlog "github.com/opentracing/opentracing-go/log"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/internal/ikiosocket"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/internal/rpcerror"
	"golang.org/x/net/context"
)

func SClient(endpoint string, options ...Option) Client {
	opts := newOptions(options...)
	return newGeneralClient(&binaryFactory{
		opts: opts,
		pool: newPool(opts.PoolSize, opts.dialer, opts.PoolTTL, nil),
	}, endpoint, opts)
}

type binaryFactory struct {
	opts Options
	pool *pool
}

func (r *binaryFactory) Name() string {
	return "binary"
}

func (r *binaryFactory) Factory(host string) (core.Plugin, error) {
	sock, err := r.pool.getSocket(host)
	if err != nil {
		return nil, err
	}

	return core.Function(func(ctx context.Context, c core.Core) {
		var (
			err    error
			span   = opentracing.SpanFromContext(ctx)
			codec  = r.opts.Codec
			rpcctx = ctx.Value(rpcContextKey).(*rpcContext)
		)
		rpcctx.host = host
		span.SetTag("proto", "rpc/binary")

		defer func() {
			ext.Error.Set(span, true)
			c.AbortErr(err)
		}()

		span.LogFields(openlog.String("event", "encode request body"))
		body, err := codec.Encode(rpcctx.Request)
		if err != nil {
			span.LogFields(
				openlog.String("event", "decode error"),
				openlog.Error(err),
			)
			rpcctx.retCode = rpcerror.Internal
			err = rpcerror.Error(rpcerror.Internal, fmt.Errorf("encode: %v", err))
			return
		}

		carrier := opentracing.TextMapCarrier{}
		if tracer := r.opts.Tracer; tracer != nil {
			tracer.Inject(span.Context(), opentracing.TextMap, carrier)
		}

		header := make(map[string]string)
		carrier.ForeachKey(func(key, value string) error {
			header[key] = value
			return nil
		})

		span.LogFields(openlog.String("event", "rpc client invoking"))
		body, err = sock.Call(rpcctx.Endpoint, header, body)
		if err == ikiosocket.ErrExited {
			// only close socket when exit
			r.pool.release(host, sock, err)
		}
		if err != nil {
			span.LogFields(
				openlog.String("event", "transport error"),
				openlog.Error(err),
			)
			rpcctx.retCode = rpcerror.Internal
			err = rpcerror.Error(rpcerror.Internal, err)
			return
		}

		span.LogFields(openlog.String("event", "decode response"))
		err = codec.Decode(body, rpcctx.Response)
		if err != nil {
			span.LogFields(
				openlog.String("event", "decode error"),
				openlog.Error(err),
			)
			rpcctx.retCode = rpcerror.Internal
			err = rpcerror.Error(rpcerror.Internal, fmt.Errorf("decode: %v", err))
			return
		}
	}), nil
}
