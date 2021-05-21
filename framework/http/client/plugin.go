package client

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/uber/jaeger-client-go"
	"github.com/yunfeiyang1916/toolkit/framework/breaker"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/ecode"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/metric"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/namespace"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/recovery"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/retry"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/sd"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/tracing"
	"github.com/yunfeiyang1916/toolkit/framework/ratelimit"
	"github.com/yunfeiyang1916/toolkit/framework/utils"
	"github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/metrics"
	"golang.org/x/net/context"
)

// core plugin encapsulation
type HandlerFunc func(c *Context)

func (p HandlerFunc) Do(ctx context.Context, flow core.Core) {
	c := ctx.Value(iCtxKey).(*Context)
	c.Ctx = ctx
	c.core = flow
	p(c)
}

func (c *client) delayPlugins() core.Plugin {
	return core.Function(func(ctx context.Context, flow core.Core) {
		c.mu.Lock()
		defer c.mu.Unlock()
		for _, v := range c.ps {
			v.Do(ctx, flow)
		}
	})
}

func (c *client) recover() core.Plugin {
	return recovery.Recovery(true)
}

func (c *client) tracing() core.Plugin {
	return core.Function(func(ctx context.Context, flow core.Core) {
		cc := ctx.Value(iCtxKey).(*Context)
		p := tracing.TraceClient(c.options.tracer, fmt.Sprintf("HTTP Client %s %s", cc.Req.method, cc.Req.path), false)
		p.Do(ctx, flow)
	})
}

func (c *client) retry() core.Plugin {
	return core.Function(func(ctx context.Context, flow core.Core) {
		times := ctx.Value(retry.Key).(int)
		retry.Retry(times).Do(ctx, flow)
	})
}

func (c *client) namespace() core.Plugin {
	return namespace.Namespace(c.options.namespace)
}

func (c *client) peername() core.Plugin {
	return metric.SDName(c.options.localName)
}

func (c *client) upstream() core.Plugin {
	if c.options.cluster != nil {
		return sd.Upstream(c, c.options.cluster)
	}
	return nil
}

func (c *client) breaker() core.Plugin {
	return core.Function(func(ctx context.Context, flow core.Core) {
		if c.options.breaker == nil {
			return
		}
		path := c.parseRequestShortPath(ctx)
		brk := c.options.breaker.GetBreaker(breaker.ClientBreakerType, c.options.namespace, c.options.serviceName, path)
		if brk == nil {
			return
		}

		if err := brk.Call(func() error {
			flow.Next(ctx)
			return flow.Err()
		}, 0); err != nil {
			flow.AbortErr(err)
		}
	})
}

func (c *client) rateLimit() core.Plugin {
	return core.Function(
		func(ctx context.Context, flow core.Core) {
			if c.options.limiter == nil {
				return
			}
			path := c.parseRequestShortPath(ctx)
			lim := c.options.limiter.GetLimiter(ratelimit.ClientLimiterType, c.options.namespace, c.options.serviceName, path)
			if lim != nil && !lim.Allow() {
				err := flow.Err()
				if err != nil {
					err = errors.Wrap(err, ratelimit.ErrLimited.Error())
				} else {
					err = ratelimit.ErrLimited
				}
				flow.AbortErr(err)
			}
		})
}

func (c *client) parseRequestShortPath(ctx context.Context) string {
	path := ctx.Value(iReqPathKey).(string)
	return strings.Split(path, "?")[0]
}

func (c *client) logging() core.Plugin {
	return core.Function(func(ctx context.Context, flow core.Core) {
		startTime := time.Now()
		defer func() {
			endTime := time.Now()
			rspCode := ecode.ConvertErr(flow.Err())
			cc := ctx.Value(iCtxKey).(*Context)
			rc := recover()
			if rc != nil {
				logging.CrashLogf("recover panic info:%q,raw request method:%s,uri:%s\n", rc, cc.Req.method, cc.Req.finalURI)
			}
			span := opentracing.SpanFromContext(ctx)
			var traceId string
			spanCtx := span.Context()
			if sc, ok := spanCtx.(jaeger.SpanContext); ok {
				traceId = sc.TraceID().String()
			}

			uri := cc.Req.path
			tags := make([]interface{}, len(cc.Req.ro.metricTags)*2)
			idx := 0
			for k, v := range cc.Req.ro.metricTags {
				tags[idx] = k
				tags[idx+1] = v
				idx++
			}
			tags = append(tags, metrics.TagCode, rspCode, "clienttag", "client")
			methodName := c.parseMetricMethodName(cc.Req.path)
			metrics.Timer("client."+methodName, startTime, tags...)

			isSlow := cc.Req.ro.slowTime > 0 && endTime.Sub(startTime) > cc.Req.ro.slowTime
			if c.options.logger.B() == nil && !isSlow { // logging disable
				if rc != nil {
					panic(rc)
				}
				return
			}
			logItems := []interface{}{
				"start", startTime.Format(utils.TimeFormat),
				"cost", math.Ceil(float64(time.Since(startTime).Nanoseconds()) / 1e6),
				"trace_id", traceId,
				"local_name", c.options.localName,
				"service_name", c.options.serviceName,
				"req_method", cc.Req.method,
				"req_uri", uri,
				"rsp_code", rspCode,
				"address", cc.Req.host,
				"namespace", namespace.GetNamespace(ctx),
			}
			if cc.Resp != nil {
				logItems = append(logItems, "http_code", cc.Resp.Code())
			}
			if flow.Err() != nil {
				logItems = append(logItems, "error", fmt.Sprintf("%q", flow.Err().Error()))
			}
			if c.options.logger.B() != nil {
				c.options.logger.B().Debugw("httpclient", logItems...)
			}
			if isSlow {
				span.SetTag("slow", true)
				logging.Log(logging.SlowLoggerName).Debugw("httpslow", logItems...)
			}
			if rc != nil {
				panic(rc)
			}
		}()
		flow.Next(ctx)
	})
}

func (c *client) parseMetricMethodName(path string) string {
	return strings.Trim(strings.Replace(strings.Split(path, "?")[0], "/", ".", -1), ".")
}

func (c *client) sender() core.Plugin {
	return core.Function(func(ctx context.Context, flow core.Core) {
		var (
			isSampled bool
			dumpReq   []byte
			dumpResp  []byte
		)
		span := opentracing.SpanFromContext(ctx)
		cc := ctx.Value(iCtxKey).(*Context)
		nReq := cc.Req.raw
		var cancelF context.CancelFunc
		if cc.Req.ro.reqTimeout > 0 {
			var nCtx context.Context
			nCtx, cancelF = context.WithTimeout(ctx, cc.Req.ro.reqTimeout)
			nReq = nReq.WithContext(nCtx)
		}
		span.SetOperationName(fmt.Sprintf("HTTP Client %s %s", nReq.Method, nReq.URL.Path))
		ext.PeerService.Set(span, c.options.serviceName)
		ext.Component.Set(span, "inkelogic/go-httpclient")

		spanCtx := span.Context()
		if sc, ok := spanCtx.(jaeger.SpanContext); ok {
			// sampling record, ignored big body
			if sc.IsSampled() && !cc.Req.bigBody {
				isSampled = true
				dumpReq, _ = httputil.DumpRequest(nReq, true)
			}
		}
		resp, err := c.client.Do(nReq)
		span.LogFields(opentracinglog.Object("HTTP Done", err))
		if err != nil {
			if strings.Contains(err.Error(), "context deadline exceeded") {
				span.LogFields(
					opentracinglog.String("event", "HTTPDo"),
					opentracinglog.String("reason", "Context DeadlineExceeded"),
					opentracinglog.Error(err))
				flow.AbortErr(context.DeadlineExceeded)
			} else if strings.Contains(err.Error(), "context canceled") {
				span.LogFields(
					opentracinglog.String("event", "HTTPDo"),
					opentracinglog.String("reason", "Context Canceled"),
					opentracinglog.Error(err))
				flow.AbortErr(context.Canceled)
			} else {
				span.LogFields(
					opentracinglog.String("event", "HTTPDo"),
					opentracinglog.String("reason", fmt.Sprintf("error: %v", err)),
					opentracinglog.Error(err))
				flow.AbortErr(err)
			}
			ext.Error.Set(span, true)

			// 请求失败,释放资源
			if cancelF != nil {
				cancelF()
			}

			// 如果是big body,出于性能考虑,框架内不做重试,需要外部实现.
			if cc.Req.bigBody {
				flow.AbortErr(retry.BreakError{Err: err})
				return
			}

			if cc.Req.ro.retryTimes > 0 {
				if cc.orgReq != nil { // 使用原始请求
					cc.Req.raw = cloneRawRequest(cc.orgCtx, cc.orgReq)
				} else { // 重建请求
					cc.Req.raw = nil
					cc.Req.buildOnce = false
				}

				// 重置body reader
				if cc.Req.body != nil && cc.Req.body.Len() > 0 {
					cc.Req.reader = bytes.NewBuffer(cc.Req.body.Bytes())
				}

				if cc.Req.raw != nil {
					cc.Req.raw.Body = ioutil.NopCloser(cc.Req.reader)
				}
			}
			return
		}

		if isSampled {
			dumpResp = utils.DumpRespBody(resp)
			span.LogFields(
				opentracinglog.String("req", utils.Base64(dumpReq)),
				opentracinglog.String("resp", utils.Base64(dumpResp)),
			)
		}

		var e error
		cc.Resp, e = BuildResp(nReq, resp)
		span.LogFields(opentracinglog.Object("BuildResp Done", e))
		if e != nil {
			flow.AbortErr(e)
			ext.Error.Set(span, true)
		}
		cc.Resp.setCancel(cancelF)
		cc.Resp.setSpan(span)
	})
}

func (c *client) buildRequest() core.Plugin {
	return core.Function(func(ctx context.Context, flow core.Core) {
		span := opentracing.SpanFromContext(ctx)
		if span != nil {
			span.LogFields(opentracinglog.String("event", "make new request"))
		}
		cc := ctx.Value(iCtxKey).(*Context)
		cc.Req.injectBaggage(ctx)
		req := cc.Req.RawRequest()
		if req == nil {
			cc.AbortErr(fmt.Errorf("make a new request fail"))
			return
		}
		nReq := tracing.ContextToHTTP(ctx, c.options.tracer, req)
		cc.Req.raw = nReq
		flow.Next(context.WithValue(nReq.Context(), iReqPathKey, cc.Req.path))
	})
}

func (c *client) Factory(host string) (core.Plugin, error) {
	return core.Function(func(ctx context.Context, flow core.Core) {
		if len(host) == 0 {
			flow.AbortErr(ecode.ErrClientLB)
		} else {
			cc := ctx.Value(iCtxKey).(*Context)
			cc.Req.host = host
		}
	}), nil
}
