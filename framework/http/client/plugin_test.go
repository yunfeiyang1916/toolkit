package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_client_parseMetricMethodName(t *testing.T) {
	c := client{}

	method := c.parseMetricMethodName("/SendMT/SendMessage")
	assert.Equal(t, "SendMT.SendMessage", method)

	method = c.parseMetricMethodName("/q?protocol=2&did=DuUfcHxoZwWfAWRkDANRMtLLrD%2BFBsFnDqMgqHhMqh3sIQcnOpzxQGkrwZjeKwhpz5JaGUst4EzqKzw77jAydiFAfff&pkg=com.meelive.ingkee")
	assert.Equal(t, "q", method)

	method = c.parseMetricMethodName("/q?protocol=2&did=DueGanBgyNdgg5hOTlSgkAPZsUu5QWb95v40x%2FZ9XKuczoaVf%2F%2FwyB2jck6PXjSYZJVhE4BYx%2BGEb14To5kgGtMQffff&pkg=com.meelive.ingkee")
	assert.Equal(t, "q", method)

	method = c.parseMetricMethodName("/v1/messaging/:id")
	assert.Equal(t, "v1.messaging.:id", method)

	method = c.parseMetricMethodName("/v1/messaging/:id?uid=991084&content=???")
	assert.Equal(t, "v1.messaging.:id", method)

	method = c.parseMetricMethodName("/")
	assert.Equal(t, "", method)

	method = c.parseMetricMethodName("/?")
	assert.Equal(t, "", method)

	method = c.parseMetricMethodName("/json/sms/g_Submit?")
	assert.Equal(t, "json.sms.g_Submit", method)

	method = c.parseMetricMethodName("")
	assert.Equal(t, "", method)
}

func Test_client_parseRequestShortPath(t *testing.T) {
	c := client{}

	ctx := context.Background()
	ctx = context.WithValue(ctx, iReqPathKey, "/user/relation/relations?uid=14489392&ids=14480600,6216696,13492198,4669287,5404436,11491971&full=1")
	path := c.parseRequestShortPath(ctx)
	assert.Equal(t, "/user/relation/relations", path)

	ctx = context.Background()
	ctx = context.WithValue(ctx, iReqPathKey, "/user/relation/numrelations?ids=7284815&aggs=fans")
	path = c.parseRequestShortPath(ctx)
	assert.Equal(t, "/user/relation/numrelations", path)

	ctx = context.Background()
	ctx = context.WithValue(ctx, iReqPathKey, "/user/relation/numrelations")
	path = c.parseRequestShortPath(ctx)
	assert.Equal(t, "/user/relation/numrelations", path)
}
