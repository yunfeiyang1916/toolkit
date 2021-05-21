package tracing

import (
	"github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/go-tls"
	"golang.org/x/net/context"
)

func TraceServer(tracer opentracing.Tracer, operationName string) core.Plugin {
	return core.Function(func(ctx context.Context, c core.Core) {
		serverSpan := opentracing.SpanFromContext(ctx)
		if serverSpan == nil {
			// All we can do is create a new root span.
			serverSpan = tracer.StartSpan(operationName)
		} else {
			serverSpan.SetOperationName(operationName)
		}
		defer serverSpan.Finish()
		otext.SpanKindRPCServer.Set(serverSpan)
		ctx = opentracing.ContextWithSpan(ctx, serverSpan)

		// Do Next plugin
		c.Next(ctx)
	})
}

func TraceClient(tracer opentracing.Tracer, operationName string, finishOnSucc bool) core.Plugin {
	return core.Function(func(ctx context.Context, c core.Core) {
		var clientSpan opentracing.Span
		var parentSpan opentracing.Span
		if parentSpan = opentracing.SpanFromContext(ctx); parentSpan == nil {
			if tlsCtx, ok := tls.GetContext(); ok {
				parentSpan = opentracing.SpanFromContext(tlsCtx)
			}
		}
		if parentSpan != nil {
			clientSpan = tracer.StartSpan(operationName, opentracing.ChildOf(parentSpan.Context()))
		} else {
			clientSpan = tracer.StartSpan(operationName)
		}
		// retry plugin needs this span instance when Next() return
		clientSpan.LogFields(opentracinglog.String("event", "ClientStart"))
		// defer clientSpan.Finish()
		otext.SpanKindRPCClient.Set(clientSpan)
		ctx = opentracing.ContextWithSpan(ctx, clientSpan)
		// Do Next plugin
		c.Next(ctx)
		if c.Err() != nil {
			clientSpan.Finish()
			return
		}
		if c.Err() == nil && finishOnSucc {
			clientSpan.Finish()
		}
	})
}
