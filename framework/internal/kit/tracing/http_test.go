package tracing

import (
	"net/http"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestContextToHTTP(t *testing.T) {
	tracer := mocktracer.New()
	tracer.RegisterExtractor(
		opentracing.HTTPHeaders, &mocktracer.TextMapPropagator{true},
	)
	contextSpan := tracer.StartSpan("testOp").(*mocktracer.MockSpan)
	ctx := opentracing.ContextWithSpan(context.Background(), contextSpan)
	req, _ := http.NewRequest("testmethod", "testurl", nil)
	ContextToHTTP(ctx, tracer, req)
	contextSpan.Finish()

	finishedSpans := tracer.FinishedSpans()
	assert.Equal(t, 1, len(finishedSpans))

	endpointSpan := finishedSpans[0]
	assert.Equal(t, "testOp", endpointSpan.OperationName)

	contextContext := contextSpan.Context().(mocktracer.MockSpanContext)
	endpointContext := endpointSpan.Context().(mocktracer.MockSpanContext)
	// ...and that the ID is unmodified.
	assert.Equal(t, contextContext.SpanID, endpointContext.SpanID)
	assert.Equal(t, 3, len(req.Header))
}

func TestHTTPToContext(t *testing.T) {
	tracer := mocktracer.New()
	tracer.RegisterExtractor(
		opentracing.HTTPHeaders, &mocktracer.TextMapPropagator{true},
	)
	req, _ := http.NewRequest("testmethod", "testurl", nil)
	ctx := HTTPToContext(tracer, req, "testOp")
	span := opentracing.SpanFromContext(ctx)
	span.Finish()
	mspan := span.(*mocktracer.MockSpan)
	assert.NotEqual(t, len(mspan.Tags()), 0)

	finishedSpans := tracer.FinishedSpans()
	assert.Equal(t, 1, len(finishedSpans))

	endpointSpan := finishedSpans[0]
	assert.Equal(t, "testOp", endpointSpan.OperationName)
}
