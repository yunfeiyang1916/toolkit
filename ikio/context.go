package ikio

import (
	"context"
)

type contextKey struct{}

var contextTimestamp = contextKey{}

type timeWrap struct {
	t    int64
	tags []interface{}
}

func SetTimestamp(ctx context.Context, t int64, tags ...interface{}) context.Context {
	return context.WithValue(ctx, contextTimestamp, timeWrap{t, tags})
}

func getTimestamp(ctx context.Context) timeWrap {
	i := ctx.Value(contextTimestamp)
	if t, ok := i.(timeWrap); ok {
		return t
	}
	return timeWrap{}
}
