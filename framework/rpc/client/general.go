package client

import (
	"fmt"
	"math"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/breaker"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/ecode"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/metric"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/namespace"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/ratelimit"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/recovery"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/retry"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/sd"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/tracing"
	"github.com/yunfeiyang1916/toolkit/framework/log"
	circuitlim "github.com/yunfeiyang1916/toolkit/framework/ratelimit"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/internal/rpcerror"
	"github.com/yunfeiyang1916/toolkit/framework/utils"
	"github.com/yunfeiyang1916/toolkit/logging"
	"golang.org/x/net/context"
)

type generalClient struct {
	defaultCore core.Core
	opts        Options
	endpoint    string
}

func newGeneralClient(f sd.Factory, endpoint string, opts Options) *generalClient {
	ps := []core.Plugin{
		// recovery
		recovery.Recovery(true),

		// retry
		retry.Retry(opts.Retries),

		// tracing
		tracing.TraceClient(opts.Tracer, fmt.Sprintf("RPC Client %s", endpoint), true),

		// local name
		metric.SDName(opts.SDName),

		// metric
		metric.Metric(fmt.Sprintf("client.%s", endpoint)),

		// slow log
		core.Function(func(ctx context.Context, c core.Core) {
			c.Next(ctx)
			rpcctx := ctx.Value(rpcContextKey).(*rpcContext)
			cost := time.Since(rpcctx.startTime)
			if cost <= opts.Slow {
				return
			}
			span := opentracing.SpanFromContext(ctx)
			span.SetTag("slow", true)

			logItems := []interface{}{
				"start", rpcctx.startTime.Format(utils.TimeFormat),
				"cost", math.Ceil(float64(time.Since(rpcctx.startTime).Nanoseconds()) / 1e6),
				"trace_id", log.TraceID(ctx)(),
				"local_name", opts.SDName,
				"service_name", opts.Name,
				"rpc_method", rpcctx.Endpoint,
				"rpc_code", rpcctx.retCode,
				"address", rpcctx.host,
				"namespace", opts.Namespace,
			}
			logging.Log(logging.SlowLoggerName).Debugw("rpcslow", logItems...)
		}),

		// business log
		core.Function(func(ctx context.Context, c core.Core) {
			span := opentracing.SpanFromContext(ctx)
			span.SetOperationName(fmt.Sprintf("RPC Client %s", endpoint))
			c.Next(ctx)
			rpcctx := ctx.Value(rpcContextKey).(*rpcContext)
			if rpcctx.retCode == rpcerror.Success {
				rpcctx.retCode = ecode.ConvertErr(ctx.Err())
			}
			if opts.Kit.B() == nil { // logging disable
				return
			}
			logItems := []interface{}{
				"start", rpcctx.startTime.Format(utils.TimeFormat),
				"cost", math.Ceil(float64(time.Since(rpcctx.startTime).Nanoseconds()) / 1e6),
				"trace_id", log.TraceID(ctx)(),
				"local_name", opts.SDName,
				"service_name", opts.Name,
				"rpc_method", rpcctx.Endpoint,
				"rpc_code", rpcctx.retCode,
				"address", rpcctx.host,
				"namespace", opts.Namespace,
			}
			if c.Err() != nil {
				logItems = append(logItems, "error", fmt.Sprintf("%q", c.Err().Error()))
			}
			opts.Kit.B().Debugw("rpcclient", logItems...)
		}),

		// rate limter
		ratelimit.Limiter(circuitlim.ClientLimiterType, opts.Namespace, opts.Name, endpoint, opts.Limiter),

		// breaker
		breaker.Breaker(opts.Namespace, opts.Name, endpoint, opts.Breaker),

		// namespace
		namespace.Namespace(opts.Namespace),

		// service discovery
		sd.Upstream(f, opts.Cluster),
	}
	defaultCore := core.New(ps)

	return &generalClient{
		opts:        opts,
		endpoint:    endpoint,
		defaultCore: defaultCore,
	}
}

func (r *generalClient) Invoke(ctx context.Context, request interface{}, response interface{}, opts ...CallOption) error {
	rpcctx := &rpcContext{
		Endpoint:  r.endpoint,
		Request:   request,
		Response:  response,
		startTime: time.Now(),
		retCode:   rpcerror.Success,
	}
	c := r.defaultCore.Copy()
	c.Next(context.WithValue(ctx, rpcContextKey, rpcctx))
	return c.Err()
}
