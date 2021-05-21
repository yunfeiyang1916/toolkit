package server

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"

	"github.com/magiconair/properties/assert"
	"github.com/yunfeiyang1916/toolkit/logging"
)

func TestContext_SetGet(t *testing.T) {
	ctx := Context{}

	ctx.Set("foo", "bar")

	val, ok := ctx.Get("foo")
	assert.Equal(t, true, ok)
	assert.Equal(t, "bar", val)

	val, ok = ctx.Get("foo2")
	assert.Equal(t, false, ok)
	assert.Equal(t, nil, val)

	now := time.Now()
	ctx.Set("foo-time", now)
	timeVal := ctx.GetTime("foo-time")
	assert.Equal(t, now, timeVal)

	duration := 10 * time.Second
	ctx.Set("foo-duration", duration)
	durationVal := ctx.GetDuration("foo-duration")
	assert.Equal(t, duration, durationVal)

	slice := []string{"1", "2", "3"}
	ctx.Set("foo-slice", slice)
	sliceVal := ctx.GetStringSlice("foo-slice")
	assert.Equal(t, slice, sliceVal)

	stringMap := map[string]interface{}{"1": "1", "2": "2", "3": "3"}
	ctx.Set("foo-map", stringMap)
	mapVal := ctx.GetStringMap("foo-map")
	assert.Equal(t, stringMap, mapVal)
}

func TestContext_MustGet(t *testing.T) {
	defer func() {
		if rc := recover(); rc != nil {
			logging.CrashLogf("TestContext_MustGet got panic, stacks:%s", string(debug.Stack()))
			val, ok := rc.(string)
			if ok && strings.Contains(val, "does not exist") {
				t.Logf("panic error:%s", val)
			} else {
				t.Fail()
			}
		}
	}()

	ctx := Context{}

	ctx.Set("foo", "bar")

	val := ctx.MustGet("foo")
	assert.Equal(t, "bar", val)

	val = ctx.MustGet("foo2")
}

func TestContext_TraceId(t *testing.T) {
	// init tracer
	cfg := jaegerconfig.Configuration{
		// SamplingServerURL: "http://localhost:5778/sampling"
		Sampler: &jaegerconfig.SamplerConfig{Type: jaeger.SamplerTypeRemote},
		Reporter: &jaegerconfig.ReporterConfig{
			LogSpans:            false,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  "127.0.0.1:6831",
		},
	}
	tracer, _, err := cfg.New("Context_TraceId")
	if err != nil {
		panic(err)
	}

	// server
	s := NewServer(Name("Context_TraceId"), Tracer(tracer))

	s.GET("/get/text", func(c *Context) {
		if c.TraceID() == "" {
			t.Fail()
		}
		c.SetBusiCode(0)
		_, _ = c.Response.WriteString("hello world")
	})

	go func() {
		err := s.Run(fmt.Sprintf(":%d", 22458))
		if err != nil {
			fmt.Println(err)
		}
	}()

	time.Sleep(time.Millisecond * 500)

	httpclient := http.Client{Timeout: 10 * time.Second}

	_, err = httpclient.Get("http://localhost:22458/get/text")
	assert.Equal(t, nil, err)
}
