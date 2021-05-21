package tracing

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/yunfeiyang1916/toolkit/logging"
	"golang.org/x/net/context"
)

// for rpc server
func BinaryToContext(tracer opentracing.Tracer, header map[string]string, operationName string, l *logging.Logger) context.Context {
	var span opentracing.Span
	wireContext, _ := tracer.Extract(opentracing.TextMap, opentracing.TextMapCarrier(header))
	span = tracer.StartSpan(operationName, ext.RPCServerOption(wireContext))
	return opentracing.ContextWithSpan(context.Background(), span)
}
