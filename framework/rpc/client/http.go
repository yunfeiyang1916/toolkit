package client

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	openlog "github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/tracing"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/internal/rpcerror"
	"github.com/yunfeiyang1916/toolkit/framework/utils"
	"golang.org/x/net/context"
)

func HClient(endpoint string, options ...Option) Client {
	opts := newOptions(options...)
	return newGeneralClient(&hclient{
		opts: opts,
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: opts.maxIdleConnsPerHost,
				MaxIdleConns:        opts.maxIdleConns,
				Proxy:               http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   opts.DialTimeout,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				DisableKeepAlives: opts.keepAlivesDisable,
			},
		},
	}, endpoint, opts)
}

type hclient struct {
	opts   Options
	client *http.Client
}

func (r *hclient) Factory(host string) (core.Plugin, error) {
	return core.Function(func(ctx context.Context, c core.Core) {
		var (
			err    error
			span   = opentracing.SpanFromContext(ctx)
			codec  = r.opts.Codec
			rpcctx = ctx.Value(rpcContextKey).(*rpcContext)
		)
		rpcctx.host = host
		span.SetTag("proto", "rpc/http")
		defer func() {
			if err == nil {
				return
			}
			ext.Error.Set(span, true)
			c.AbortErr(err)
		}()

		span.LogFields(openlog.String("event", "decode request body"))
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

		urlhost := "http://" + host + "/" + strings.Replace(rpcctx.Endpoint, ".", "/", -1)
		request, err := http.NewRequest("POST", urlhost, bytes.NewReader(body))
		if err != nil {
			span.LogFields(
				openlog.String("event", "make request error"),
				openlog.Error(err),
			)
			rpcctx.retCode = rpcerror.Internal
			err = rpcerror.Error(rpcerror.Internal, err)
			return
		}
		// record request
		var isSampled bool
		var dumpReq []byte
		spanCtx := span.Context()
		// record HttpRPC request
		if sc, ok := spanCtx.(jaeger.SpanContext); ok {
			// sampling record
			if sc.IsSampled() {
				isSampled = true
				dumpReq, _ = httputil.DumpRequest(request, true)
			}
		}
		nReq := tracing.ContextToHTTP(ctx, r.opts.Tracer, request)
		span.LogFields(openlog.String("event", "rpc client httpdo"))
		response, err := r.client.Do(nReq)
		if err != nil {
			if e, ok := err.(*url.Error); ok && e.Timeout() {
				rpcctx.Retry()
				rpcctx.retCode = rpcerror.Timeout
				err = rpcerror.Error(rpcerror.Timeout, err)
			}
			span.LogFields(
				openlog.String("event", "httpdo error"),
				openlog.Error(err),
			)
			return
		}

		defer response.Body.Close()

		span.LogFields(openlog.String("event", "read response body"))
		body, err = ioutil.ReadAll(response.Body)
		if err != nil {
			span.LogFields(
				openlog.String("event", "read response error"),
				openlog.Error(err),
			)
			rpcctx.retCode = rpcerror.Internal
			err = rpcerror.Error(rpcerror.Internal, err)
			return
		}
		if isSampled {
			// record req&resp
			span.LogFields(
				openlog.String("req", utils.Base64(dumpReq)),
				openlog.String("resp", utils.Base64(body)),
			)
		}

		if response.StatusCode != 200 {
			err = rpcerror.HTTP(body)
			span.LogFields(
				openlog.String("event", "response error"),
				openlog.Error(err),
			)
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
