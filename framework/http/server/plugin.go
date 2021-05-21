package server

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/yunfeiyang1916/toolkit/framework/breaker"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/ecode"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/namespace"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/recovery"
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

func (s *server) recover() core.Plugin {
	return recovery.Recovery(s.options.recoverPanic)
}

const internalReqBodyLogTag = "req_body"
const internalRespBodyLogTag = "resp_body"

func (s *server) namespaceKey() core.Plugin {
	return core.Function(func(ctx context.Context, flow core.Core) {
		cc := ctx.Value(iCtxKey).(*Context)
		key := namespace.GetNamespaceKey(cc.Namespace)
		if key == nil {
			flow.AbortErr(ecode.ErrUnknownNamespace)
		}
	})
}

func (s *server) logging() core.Plugin {
	return core.Function(func(ctx context.Context, flow core.Core) {
		defer func() {
			cc := ctx.Value(iCtxKey).(*Context)
			rc := recover()
			if rc != nil {
				logging.CrashLogf("recover panic info:%q,raw request method:%s,uri:%s\n", rc, cc.Request.Method, cc.Request.URL.String())
			}
			if s.options.logger.A() == nil { // logging disable
				if rc != nil {
					panic(rc)
				}
				return
			}
			code := cc.Response.Status()
			if err := flow.Err(); err != nil {
				code = ecode.ConvertHttpStatus(err)
			}

			logItems := []interface{}{
				"start", cc.startTime.Format(utils.TimeFormat),
				"cost", math.Ceil(float64(time.Since(cc.startTime).Nanoseconds()) / 1e6),
				"trace_id", cc.traceId,
				"peer_name", cc.Peer,
				"req_method", cc.Request.Method,
				"req_uri", cc.Request.URL.String(),
				"real_ip", getRemoteIP(cc.Request),
				"http_code", code,
				"busi_code", cc.BusiCode(),
				"namespace", cc.Namespace,
			}

			_ = cc.ForeachBaggage(func(key, val string) error {
				logItems = append(logItems, key[len(utils.DaeBaggageHeaderPrefix):], val)
				return nil
			})

			if flow.Err() != nil {
				logItems = append(logItems, "error", fmt.Sprintf("%q", flow.Err().Error()))
			}

			// request body 全局打印开关与单个uri接口开关
			if !s.options.reqBodyLogOff && cc.printReqBody {
				if _, ok := cc.loggingExtra[internalReqBodyLogTag]; !ok {
					logItems = append(logItems, "req_body", fmt.Sprintf("%q", cc.bodyBuff.Bytes()))
				}
			}
			// response body
			if cc.printRespBody {
				if _, ok := cc.loggingExtra[internalRespBodyLogTag]; !ok {
					logItems = append(logItems, "resp_body", fmt.Sprintf("%q", cc.Response.ByteBody()))
				}
			}

			if len(cc.loggingExtra) > 0 {
				extraList := make([]interface{}, 0)
				for k, v := range cc.loggingExtra {
					extraList = append(extraList, k, v)
				}
				if len(extraList) > 0 {
					logItems = append(logItems, extraList...)
				}
			}
			s.options.logger.A().Debugw("httpserver", logItems...)
			if rc != nil {
				panic(rc)
			}
		}()
		flow.Next(ctx)
	})
}

func (s *server) traceIDHeader() core.Plugin {
	return core.Function(func(ctx context.Context, flow core.Core) {
		cc := ctx.Value(iCtxKey).(*Context)
		cc.Response.Header().Set("X-Trace-Id", cc.TraceID())
	})
}

func (s *server) metric() core.Plugin {
	return core.Function(func(ctx context.Context, flow core.Core) {
		flow.Next(ctx)
		cc := ctx.Value(iCtxKey).(*Context)
		// 优先使用用户设置的busiCode
		code := cc.BusiCode()
		if err := flow.Err(); err != nil {
			code = int32(ecode.ConvertErr(flow.Err()))
			cc.SetBusiCode(code)
		}
		methodName := strings.Trim(strings.Replace(cc.Path, "/", ".", -1), ".")
		metricPrefix := fmt.Sprintf("RestServe.%s", methodName)
		metricPeerPrefix := fmt.Sprintf("RestServePeer.%s", methodName)
		metrics.Timer(metricPeerPrefix, cc.startTime, metrics.TagCode, code, "peer", cc.Peer)
		if cc.Namespace != "" {
			metrics.Timer(metricPrefix, cc.startTime, metrics.TagCode, code, "namespace", cc.Namespace)
		} else {
			metrics.Timer(metricPrefix, cc.startTime, metrics.TagCode, code)
		}
		cc.LoggingExtra("method_name", methodName)
	})
}

func (s *server) rateLimit() core.Plugin {
	return core.Function(func(ctx context.Context, flow core.Core) {
		if s.options.limiter == nil {
			return
		}
		cc := ctx.Value(iCtxKey).(*Context)
		lim := s.options.limiter.GetLimiter(ratelimit.ServerLimiterType, cc.Namespace, cc.Peer, cc.Path)
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

func (s *server) breaker(path string) core.Plugin {
	return core.Function(func(ctx context.Context, flow core.Core) {
		if s.options.breaker == nil {
			return
		}

		cc := ctx.Value(iCtxKey).(*Context)
		brk := s.options.breaker.GetBreaker(breaker.ServerBreakerType, cc.Namespace, "", path)
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

func (s *server) methodNotAllowed(ctx *Context) bool {
	// 405
	t := s.trees
	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method == ctx.Request.Method {
			continue
		}
		root := t[i].root
		// plugin, urlparam, found, matchPath expression
		flow, _, _, _ := root.getValue(ctx.Request.URL.Path, ctx.Params, false)
		if flow != nil {
			return true
		}
	}
	return false
}

// Deprecated: PrintRespBody func should not use anymore.
// Use PrintBodyLog func instead
func PrintRespBody(b bool) HandlerFunc {
	return func(c *Context) {
		c.printRespBody = b
	}
}

func PrintBodyLog(printReq, printResp bool) HandlerFunc {
	return func(c *Context) {
		c.printReqBody = printReq
		c.printRespBody = printResp
	}
}
