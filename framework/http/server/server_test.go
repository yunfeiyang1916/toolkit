package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
	"github.com/yunfeiyang1916/toolkit/framework/http/client"
	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/upstream"
	"github.com/yunfeiyang1916/toolkit/logging"
)

type jsonTestObj struct {
	DMErr  int    `json:"dm_error"`
	ErrMsg string `json:"error_msg"`
	Data   struct {
		Action string
	} `json:"data"`
}

func InitTestServer(handlers map[string]HandlerFunc, serverPort int) {
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
	tracer, _, err := cfg.New("danerys.test.service")
	if err != nil {
		panic(err)
	}

	// server
	s := NewServer(Name("danerys.test.service"), Tracer(tracer))

	for k, v := range handlers {
		s.ANY(k, v)
	}

	go func() {
		err := s.Run(fmt.Sprintf(":%d", serverPort))
		if err != nil {
			fmt.Println(err)
		}
	}()

	time.Sleep(1 * time.Second)
}

func TestHttpServer_WriteHeader(t *testing.T) {
	getJson := jsonTestObj{
		DMErr:  0,
		ErrMsg: "操作成功",
		Data:   struct{ Action string }{Action: "get json"},
	}

	InitTestServer(
		map[string]HandlerFunc{
			"/json/get/500": func(c *Context) {
				c.Response.WriteHeader(500)
				c.Raw(getJson, 500)
				return
			},
			"/json/get/502": func(c *Context) {
				c.Response.WriteHeader(502)
				c.JSON(map[string]interface{}{"action": "get json"}, nil)
				return
			},
			"/json/get/400": func(c *Context) {
				c.Response.WriteHeader(400)
				c.SetBusiCode(400)
				b, _ := json.Marshal(getJson)
				_, _ = c.Response.Write(b)
				return
			},
			"/json/post/403": func(c *Context) {
				c.Response.WriteHeader(403)
				c.SetBusiCode(403)
				_, _ = c.Response.WriteString("hello world 403")
				return
			},
			"/json/post/200": func(c *Context) {
				c.Response.WriteHeader(200)
				c.JSONAbort(map[string]interface{}{"action": "post json"}, nil)
				return
			},
			"/add/header": func(c *Context) {
				c.Response.Header().Add("x-my-header-1", "hello world 1")
				c.Response.Header().Add("x-my-header-2", "hello world 2")
				c.Response.Header().Add("x-my-header-3", "hello world 3")
				c.Response.WriteHeader(200)
				_, _ = c.Response.WriteString("add hello world header")
				return
			},
		},
		22356,
	)

	httpclient := http.Client{Timeout: 10 * time.Second}

	fmt.Println("========= 500 header =========")
	// 500 status
	rsp, err := httpclient.Get("http://localhost:22356/json/get/500")
	assert.Equal(t, nil, err)
	if rsp == nil {
		t.Fail()
	}
	respB, err := ioutil.ReadAll(rsp.Body)
	assert.Equal(t, nil, err)
	jsonGet500Response := jsonTestObj{}
	err = json.Unmarshal(respB, &jsonGet500Response)
	assert.Equal(t, nil, err)
	assert.Equal(t, getJson, jsonGet500Response)

	assert.Equal(t, "500 Internal Server Error", rsp.Status)

	fmt.Println("========= 502 header =========")
	// 502 status
	rsp, err = httpclient.Get("http://localhost:22356/json/get/502")
	assert.Equal(t, nil, err)
	if rsp == nil {
		t.Fail()
	}
	respB, err = ioutil.ReadAll(rsp.Body)
	assert.Equal(t, nil, err)
	jsonGet502Response := jsonTestObj{}
	err = json.Unmarshal(respB, &jsonGet502Response)
	assert.Equal(t, nil, err)
	assert.Equal(t, jsonTestObj{DMErr: 0, ErrMsg: "0", Data: struct{ Action string }{Action: "get json"}}, jsonGet502Response)

	assert.Equal(t, "502 Bad Gateway", rsp.Status)

	fmt.Println("========= 400 header =========")
	// 400 status
	rsp, err = httpclient.Get("http://localhost:22356/json/get/400")
	assert.Equal(t, nil, err)
	if rsp == nil {
		t.Fail()
	}
	respB, err = ioutil.ReadAll(rsp.Body)
	assert.Equal(t, nil, err)
	jsonGet400Response := jsonTestObj{}
	err = json.Unmarshal(respB, &jsonGet400Response)
	assert.Equal(t, nil, err)
	assert.Equal(t, getJson, jsonGet400Response)

	assert.Equal(t, "400 Bad Request", rsp.Status)

	fmt.Println("========= 403 header =========")
	// 403 status
	rsp, err = httpclient.Post("http://localhost:22356/json/post/403", "Content-Type: application/json; charset=utf-8", nil)
	assert.Equal(t, nil, err)
	if rsp == nil {
		t.Fail()
	}
	respB, err = ioutil.ReadAll(rsp.Body)
	assert.Equal(t, nil, err)
	assert.Equal(t, "hello world 403", string(respB[:]))

	assert.Equal(t, "403 Forbidden", rsp.Status)

	fmt.Println("========= 200 header =========")
	// 200 status
	rsp, err = httpclient.Post("http://localhost:22356/json/post/200", "Content-Type: application/json; charset=utf-8", nil)
	assert.Equal(t, nil, err)
	if rsp == nil {
		t.Fail()
	}
	respB, err = ioutil.ReadAll(rsp.Body)
	assert.Equal(t, nil, err)
	jsonPost200Response := jsonTestObj{}
	err = json.Unmarshal(respB, &jsonPost200Response)
	assert.Equal(t, nil, err)
	assert.Equal(t, jsonTestObj{DMErr: 0, ErrMsg: "0", Data: struct{ Action string }{Action: "post json"}}, jsonPost200Response)

	fmt.Println("========= add header =========")
	// add header
	rsp, err = httpclient.Post("http://localhost:22356/add/header", "Content-Type: application/json; charset=utf-8", nil)
	assert.Equal(t, nil, err)
	if rsp == nil {
		t.Fail()
	}
	respB, err = ioutil.ReadAll(rsp.Body)
	assert.Equal(t, nil, err)
	assert.Equal(t, "add hello world header", string(respB[:]))

	assert.Equal(t, "200 OK", rsp.Status)
	assert.Equal(t, "hello world 1", rsp.Header.Get("x-my-header-1"))
	assert.Equal(t, "hello world 2", rsp.Header.Get("x-my-header-2"))
	assert.Equal(t, "hello world 3", rsp.Header.Get("x-my-header-3"))
	if rsp.Header.Get("X-Trace-Id") == "" {
		t.Fail()
	}
}

func TestHttpServer_TraceId(t *testing.T) {

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
	tracer, _, err := cfg.New("danerys.test.service")
	if err != nil {
		panic(err)
	}

	// server
	s := NewServer(Name("danerys.test.service"), Tracer(tracer))

	s.GET("/get/text", func(c *Context) {
		c.SetBusiCode(0)
		_, _ = c.Response.WriteString("hello world")
	})

	go func() {
		err := s.Run(fmt.Sprintf(":%d", 22358))
		if err != nil {
			fmt.Println(err)
		}
	}()

	time.Sleep(1 * time.Second)

	httpclient := http.Client{Timeout: 10 * time.Second}

	// 404
	rsp, err := httpclient.Get("http://localhost:22358/get/text/404")
	assert.Equal(t, nil, err)
	assert.Equal(t, "404 Not Found", rsp.Status)

	traceId := rsp.Header.Get("X-Trace-Id")
	t.Logf("traceId:%s\n", traceId)
	if traceId == "" {
		t.Fail()
	}

	// 405
	rsp, err = httpclient.Post("http://localhost:22358/get/text", "Content-Type: application/json; charset=utf-8", nil)
	assert.Equal(t, nil, err)
	assert.Equal(t, "405 Method Not Allowed", rsp.Status)

	traceId = rsp.Header.Get("X-Trace-Id")
	t.Logf("traceId:%s\n", traceId)
	if traceId == "" {
		t.Fail()
	}

	// 200
	rsp, err = httpclient.Get("http://localhost:22358/get/text")
	assert.Equal(t, nil, err)
	assert.Equal(t, "200 OK", rsp.Status)

	traceId = rsp.Header.Get("X-Trace-Id")
	t.Logf("traceId:%s\n", traceId)
	if traceId == "" {
		t.Fail()
	}
}

func TestHttpServer_HeaderWithoutResponse(t *testing.T) {
	cfg := jaegerconfig.Configuration{
		Sampler: &jaegerconfig.SamplerConfig{Type: jaeger.SamplerTypeRemote},
		Reporter: &jaegerconfig.ReporterConfig{
			LogSpans:            false,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  "127.0.0.1:6831",
		},
	}
	tracer, _, err := cfg.New("danerys.test.service")
	if err != nil {
		panic(err)
	}

	// server
	s := NewServer(Name("danerys.test.service"), Tracer(tracer))

	s.GET("/get/text/404", func(c *Context) {
		c.SetBusiCode(0)
		c.Response.WriteHeader(404)
		c.Response.Header().Set("x-header-1", "header-1")
		c.Response.Header().Set("x-header-2", "header-2")
		c.Response.Header().Set("x-header-3", "header-3")
	})

	s.GET("/get/text/500", func(c *Context) {
		c.SetBusiCode(0)
		c.Response.WriteHeader(500)
		c.Response.Header().Set("x-header-1", "header-1")
		c.Response.Header().Set("x-header-2", "header-2")
		c.Response.Header().Set("x-header-3", "header-3")
	})

	s.GET("/get/text/200", func(c *Context) {
		c.SetBusiCode(0)
		c.Response.WriteHeader(200)
		c.Response.Header().Set("x-header-1", "header-1")
		c.Response.Header().Set("x-header-2", "header-2")
		c.Response.Header().Set("x-header-3", "header-3")
	})

	go func() {
		err := s.Run(fmt.Sprintf(":%d", 22368))
		if err != nil {
			fmt.Println(err)
		}
	}()

	time.Sleep(1 * time.Second)

	// 404
	httpclient := http.Client{Timeout: 10 * time.Second}
	rsp, err := httpclient.Get("http://localhost:22368/get/text/404")
	assert.Equal(t, nil, err)
	assert.Equal(t, "404 Not Found", rsp.Status)
	assert.Equal(t, "header-1", rsp.Header.Get("x-header-1"))
	assert.Equal(t, "header-2", rsp.Header.Get("x-header-2"))
	assert.Equal(t, "header-3", rsp.Header.Get("x-header-3"))
	if rsp.Header.Get("X-Trace-Id") == "" {
		t.Fail()
	}

	// 500
	rsp, err = httpclient.Get("http://localhost:22368/get/text/500")
	assert.Equal(t, nil, err)
	assert.Equal(t, "500 Internal Server Error", rsp.Status)
	assert.Equal(t, "header-1", rsp.Header.Get("x-header-1"))
	assert.Equal(t, "header-2", rsp.Header.Get("x-header-2"))
	assert.Equal(t, "header-3", rsp.Header.Get("x-header-3"))
	if rsp.Header.Get("X-Trace-Id") == "" {
		t.Fail()
	}

	// 200
	rsp, err = httpclient.Get("http://localhost:22368/get/text/200")
	assert.Equal(t, nil, err)
	assert.Equal(t, "200 OK", rsp.Status)
	assert.Equal(t, "header-1", rsp.Header.Get("x-header-1"))
	assert.Equal(t, "header-2", rsp.Header.Get("x-header-2"))
	assert.Equal(t, "header-3", rsp.Header.Get("x-header-3"))
	if rsp.Header.Get("X-Trace-Id") == "" {
		t.Fail()
	}
}

func TestServer_ioCopyWriter(t *testing.T) {
	cfg := jaegerconfig.Configuration{
		Sampler: &jaegerconfig.SamplerConfig{Type: jaeger.SamplerTypeRemote},
		Reporter: &jaegerconfig.ReporterConfig{
			LogSpans:            false,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  "127.0.0.1:6831",
		},
	}
	tracer, _, err := cfg.New("danerys.test.service")
	if err != nil {
		panic(err)
	}

	// server
	s := NewServer(Name("danerys.test.service"), Tracer(tracer))
	s.GET("/io/copy/200", func(c *Context) {
		c.SetBusiCode(0)
		c.Response.WriteHeader(200)
		c.Response.Header().Set("x-header-1", "header-1")
		c.Response.Header().Set("x-header-2", "header-2")
		c.Response.Header().Set("x-header-3", "header-3")
		_, _ = io.Copy(c.Response.Writer(), bytes.NewReader([]byte("hello world")))
	})

	go func() {
		err := s.Run(fmt.Sprintf(":%d", 22378))
		if err != nil {
			fmt.Println(err)
		}
	}()

	time.Sleep(1 * time.Second)

	// 200
	httpclient := http.Client{Timeout: 10 * time.Second}

	rsp, err := httpclient.Get("http://localhost:22378/io/copy/200")
	assert.Equal(t, nil, err)
	assert.Equal(t, "200 OK", rsp.Status)
	assert.Equal(t, "header-1", rsp.Header.Get("x-header-1"))
	assert.Equal(t, "header-2", rsp.Header.Get("x-header-2"))
	assert.Equal(t, "header-3", rsp.Header.Get("x-header-3"))
	if rsp.Header.Get("X-Trace-Id") == "" {
		t.Fail()
	}

	bodyB, err := ioutil.ReadAll(rsp.Body)
	assert.Equal(t, nil, err)
	assert.Equal(t, "hello world", string(bodyB))
}

func TestHttpServer_ServeHTTPPanic(t *testing.T) {

	defer func() {
		if rc := recover(); rc != nil {
			logging.CrashLogf("TestHttpServer_ServeHTTPPanic got panic, stacks:%q", debug.Stack())
			debug.PrintStack()
		}
	}()

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
	tracer, _, err := cfg.New("abc")
	if err != nil {
		panic(err)
	}

	// server
	s := NewServer(Port(22233), Name("a.b.c"), Tracer(tracer))

	s.ANY("/hello/panic", func(c *Context) {
		var nilHello map[string]string
		nilHello["hahaha"] = "123"
		_, _ = c.Response.WriteString(nilHello["hahaha"])
	})

	go func() {
		err := s.Run(":22244")
		if err != nil {
			fmt.Println(err)
		}
	}()
	time.Sleep(1 * time.Second)

	tests := map[string]struct {
		request  *http.Request
		response *http.Response
		expect   interface{}
	}{
		"call10": {
			request:  httptest.NewRequest("GET", "http://localhost:22244/hello/panic", nil),
			response: &http.Response{},
			expect:   "nil rsp",
		},
	}

	wg := sync.WaitGroup{}

	for _, rr := range tests {
		tt := rr
		wg.Add(1)
		go func() {
			wg.Done()
			b := doclient(tt.request.Method, tt.request.URL.String(), nil)
			assert.Equal(t, tt.expect, string(b))
		}()
	}
	wg.Wait()

	time.Sleep(1 * time.Second)
	_ = s.Stop()

}

func TestHttpServer_ServeHTTP(t *testing.T) {
	s := NewServer(Name("a.b.c"))
	s.GET("/hello", func(c *Context) {
		_, _ = c.Response.WriteString("hello")
		c.Next()
	}, func(c *Context) {
		_, _ = c.Response.WriteString(" world")
	})

	s.GET("/user/:name", func(c *Context) {
		v := c.Params.ByName("name")
		str := fmt.Sprintf("hello %s", v)
		_, _ = c.Response.WriteString(str)
	})

	v1 := s.GROUP("/v1")
	{
		v1.GET("/login", func(c *Context) {
			_, _ = c.Response.WriteString("welcome")
		})
	}

	go func() {
		err := s.Run(":22244")
		if err != nil {
			fmt.Println(err)
		}
	}()
	time.Sleep(1 * time.Second)

	tests := map[string]struct {
		request  *http.Request
		response *http.Response
		expect   interface{}
	}{
		"call1": {
			request:  httptest.NewRequest("GET", "http://localhost:22244/hello?a=b", nil),
			response: &http.Response{},
			expect:   "hello world",
		},
		"call2": {
			request:  httptest.NewRequest("GET", "http://localhost:22244/user/jack", nil),
			response: &http.Response{},
			expect:   "hello jack",
		},
		"call3": {
			request:  httptest.NewRequest("GET", "http://localhost:22244/user/jack23", nil),
			response: &http.Response{},
			expect:   "hello jack23",
		},
		"call4": {
			request:  httptest.NewRequest("GET", "http://localhost:22244/user/jack24", nil),
			response: &http.Response{},
			expect:   "hello jack24",
		},
		"call5": {
			request:  httptest.NewRequest("GET", "http://localhost:22244/user/jack25", nil),
			response: &http.Response{},
			expect:   "hello jack25",
		},
		"call6": {
			request:  httptest.NewRequest("GET", "http://localhost:22244/user/jack26", nil),
			response: &http.Response{},
			expect:   "hello jack26",
		},
		"call7": {
			request:  httptest.NewRequest("GET", "http://localhost:22244/user/jack22", nil),
			response: &http.Response{},
			expect:   "hello jack22",
		},
		"call8": {
			request:  httptest.NewRequest("GET", "http://localhost:22244/v1/login", nil),
			response: &http.Response{},
			expect:   "welcome",
		},
		"call9": {
			request:  httptest.NewRequest("GET", "http://localhost:22244/root", nil),
			response: &http.Response{},
			expect:   "Not Found",
		},
	}

	wg := sync.WaitGroup{}

	for _, rr := range tests {
		tt := rr
		wg.Add(1)
		go func() {
			wg.Done()
			b := doclient(tt.request.Method, tt.request.URL.String(), nil)
			assert.Equal(t, tt.expect, string(b))
		}()
	}
	wg.Wait()

	time.Sleep(1 * time.Second)
	_ = s.Stop()
}

func doclient(method string, url string, body io.Reader) []byte {
	clusterName := "test_client"
	config := config.NewCluster()
	config.Name = clusterName
	config.StaticEndpoints = "localhost:22233"
	manager := upstream.NewClusterManager()
	_ = manager.InitService(config)
	//nc := client.NewClient(client.Cluster(manager.Cluster(clusterName)))
	nc := client.NewClient()
	req := client.NewRequest()
	req.WithMethod(method)
	req.WithURL(url)
	req.WithBody(body)
	var rsp *client.Response
	var err error
	if rsp, err = nc.Call(req); err != nil {
		if rsp != nil && rsp.Code() == 404 {
			return []byte("Not Found")
		}
	}
	if rsp != nil {
		fmt.Println("rsp:", rsp.String(), "http code:", rsp.Code(), "header:", rsp.GetHeader("X-Trace-Id"))
		return rsp.Bytes()
	}
	return []byte("nil rsp")
}

func TestHttpServer_ServeHTTP_continue(t *testing.T) {
	s := NewServer(Port(22232), Name("a.b.c"))
	s.GET("/hello", func(c *Context) {
		c.Response.WriteString("hello")
		c.Next()
	}, func(c *Context) {
		c.Response.WriteString(" world")
	})

	go func() {
		err := s.Run()
		if err != nil {
			fmt.Println(err)
		}
	}()
	time.Sleep(1 * time.Second)

	//client
	b := doclient("GET", "http://localhost:22232/hello", nil)
	assert.Equal(t, "hello world", string(b))
	time.Sleep(1 * time.Second)
	fmt.Println()
	s.Stop()
}

type Input struct {
	Message string `json:"aaa"`
}

func TestHttpServer_ServeHTTP_Abort(t *testing.T) {
	s := NewServer(Logger(nil), Port(22236), Name("a.b.c"))
	s.GET("/hello", func(c *Context) {
		c.Response.WriteString("hello")

		input := &Input{}
		_ = input
		var err error
		decoder := json.NewDecoder(c.Request.Body)
		err = decoder.Decode(&input)
		bb, _ := ioutil.ReadAll(c.Request.Body)
		fmt.Println("input:", input, "bb:", string(bb), err)

		c.Abort()
	}, func(c *Context) {
		c.Response.WriteString(" world")
	})

	go func() {
		err := s.Run()
		if err != nil {
			fmt.Println(err)
		}
	}()
	time.Sleep(1 * time.Second)

	//client
	buf := bytes.NewBufferString(`{"aaa":"bbb"}`)
	b := doclient("GET", "http://localhost:22236/hello", buf)
	assert.Equal(t, "hello", string(b))
	time.Sleep(1 * time.Second)
	s.Stop()
}

func TestHttpServer_ServeHTTP_Abort_error(t *testing.T) {
	s := NewServer(Port(32234), Name("a.b.c"))
	s.GET("/hello", func(c *Context) {
		c.Response.WriteString("hello")
		c.Abort()
		err := fmt.Errorf("force abort")
		c.AbortErr(err)
		if c.Err().Error() == "force abort" {
			fmt.Println("force abort!!!!!")
		}
	}, func(c *Context) {
		c.Response.WriteString(" world")
	})

	go func() {
		err := s.Run()
		if err != nil {
			fmt.Println(err)
		}
	}()
	time.Sleep(1 * time.Second)

	//client
	b := doclient("GET", "http://localhost:32234/hello?a=b&c=d", nil)
	assert.Equal(t, "helloforce abort", string(b))
	time.Sleep(1 * time.Second)
	s.Stop()
}

func TestServer_Stop(t *testing.T) {
	s := NewServer(Name("a.b.c"))
	go func() {
		err := s.Run(":22345")
		if err != nil {
			panic(err)
		}
	}()
	time.Sleep(1 * time.Second)
	s.Stop()
}

func TestResponseWriter_WriteJson(t *testing.T) {
	s := NewServer(Name("a.b.c"))
	s.GET("/xxx", func(c *Context) {
		http.Error(c.Response.Writer(), "internal-500", http.StatusInternalServerError)
		a := map[string]string{"aa": "bb", "cc": "dd"}
		c.Response.WriteJSON(a)
		c.Response.WriteHeader(200)
		c.Response.WriteHeader(200)
		c.Response.WriteHeader(200)
		c.Response.WriteHeader(200)
		http.Error(c.Response.Writer(), "internal-500", 500)
		c.Response.WriteHeader(200)
		c.Response.WriteHeader(200)
	})

	go func() {
		s.Run(":11111")
	}()
	time.Sleep(2 * time.Second)

	//client
	doclient("GET", "http://localhost:11111/xxx", nil)
}
