package logging

import (
	"fmt"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
)

type contextFunc func(ctx context.Context) (string, string)

var contextList []contextFunc

func RegisteCtx(cb contextFunc) {
	contextList = append(contextList, cb)
}

func extraTraceID(ctx context.Context) string {
	span := opentracing.SpanFromContext(ctx)
	var traceID string
	if span == nil {
		traceID = ""
	} else {
		traceID = strings.SplitN(fmt.Sprintf("%s", span.Context()), ":", 2)[0]
	}
	return traceID
}
