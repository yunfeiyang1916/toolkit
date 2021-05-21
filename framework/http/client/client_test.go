package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/uber/jaeger-client-go"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
	"github.com/yunfeiyang1916/toolkit/framework/http/server"
	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/upstream"
	"github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/rolling"
)

func TestClient_Call(t *testing.T) {
	// server
	s := server.NewServer(server.Name("a.b.c"))
	s.GET("/hello", func(c *server.Context) {
		c.Response.WriteString("hello world")
	})
	s.POST("/user/:name", func(c *server.Context) {
		v := c.Params.ByName("name")
		str := fmt.Sprintf("hello %s", v)
		c.Response.WriteString(str)
	})
	// start
	go func() {
		err := s.Run(":28081")
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(1 * time.Second)

	uri := "http://localhost:28081/hello"
	r, _ := http.NewRequest("GET", uri, nil)

	req2 := NewRequest(context.Background()).WithMethod("POST").WithURL("http://localhost:28081/user/jack")
	req2.build()

	req3 := NewRequest(context.Background()).WithMethod("POST").WithURL("http://localhost:28081/user/tom")
	req3.build()

	tests := map[string]struct {
		request  *http.Request
		response *http.Response
		expect   interface{}
	}{
		"call1": {
			request: r,
			// request:  NewRequest().WithMethod("GET").WithURL("http://localhost:28081").WithPath("/hello").RawRequest(),
			response: &http.Response{},
			expect:   "hello world",
		},
		"call2": {
			request:  req2.RawRequest(),
			response: &http.Response{},
			expect:   "hello jack",
		},
		"call3": {
			// request:  httptest.NewRequest("POST", "http://localhost:28081/user/tom", nil),
			request:  req3.RawRequest(),
			response: &http.Response{},
			expect:   "hello tom",
		},
	}

	nc := NewClient()
	nc.Use(func(c *Context) {
		fmt.Println("***************")
	})
	for _, tt := range tests {
		rr := tt.request
		req := NewRequest(context.Background())
		req.AddHeader("key", "value")
		req.WithRequest(rr)
		var rsp *Response
		var err error
		if rsp, err = nc.Call(req); err != nil {
			panic(err)
		}
		assert.Equal(t, tt.expect, rsp.String())
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)
	s.Stop()

}

type Content struct {
	Text string `json:"text"`
}

func TestClientOptions(t *testing.T) {
	// server
	s := server.NewServer(server.Port(32444), server.Name("a.b.c"))
	// register
	s.POST("/v1", func(c *server.Context) {
		time.Sleep(3 * time.Second)
		c.Response.WriteString("{\"text\":\"hello v1\"}")

	})
	// start
	go func() {
		err := s.Run()
		if err != nil {
			fmt.Println(err)
		}
	}()
	time.Sleep(1 * time.Second)

	// cluster
	clusterName := "test_client"
	config := config.NewCluster()
	config.Name = clusterName
	config.StaticEndpoints = "localhost:32444"
	manager := upstream.NewClusterManager()
	manager.InitService(config)

	nc2 := NewClient(
		Cluster(manager.Cluster(clusterName)),
		RetryTimes(1),
		Tracer(opentracing.GlobalTracer()),
		DialTimeout(30*time.Second),
		IdleConnTimeout(10*time.Second),
		KeepAliveTimeout(10*time.Second),
		MaxIdleConns(100),
		MaxIdleConnsPerHost(10),
		RequestTimeout(5*time.Second),
		SlowTimeout(3*time.Second),
	)
	req2 := NewRequest(context.Background()).
		WithPath("/v1").WithMethod("POST")
	_, err := nc2.Call(req2)
	if err != nil {
		panic(err)
	}
	time.Sleep(1 * time.Second)
	s.Stop()
}

func TestResponse(t *testing.T) {
	// server
	s := server.NewServer(server.Port(32345), server.Name("a.b.c"))
	// register
	s.POST("/v1", func(c *server.Context) {
		c.Response.WriteString("{\"text\":\"hello v1\"}")
	})
	// start
	go func() {
		err := s.Run()
		if err != nil {
			panic(err)
		}
	}()
	time.Sleep(1 * time.Second)

	nc2 := NewClient()
	ro := &RequestOption{}
	ro.RetryTimes(1).RequestTimeoutMS(5000)
	req2 := NewRequest(context.Background()).
		WithOption(ro).
		WithMethod("post").
		WithURL("http://localhost:32345/v1").
		WithBody(bytes.NewBuffer([]byte(`hello world`)))
	rsp2, err := nc2.Call(req2)
	if err != nil {
		panic(err)
	}

	file := "./a.txt"
	rsp2.Save(file)
	defer os.Remove(file)
	time.Sleep(1 * time.Second)
	s.Stop()
}

func TestDefault(t *testing.T) {

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

	s := server.NewServer(server.Port(54321), server.Name("a.b.c"), server.Tracer(tracer))
	s.GET("/root/bb", func(c *server.Context) {

		nctx := c.SetBaggage("kkkkk", "vvvvvv")
		fmt.Println("on root bb")
		nc := NewClient()
		req := NewRequest(nctx).WithURL("http://localhost:54322/root/aa")
		req.SetBaggage("mmmmm", "nnnnn")
		nc.Call(req)

		c.Response.WriteString("hello root bb")

	})
	// start
	go func() {
		err := s.Run()
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(1 * time.Second)

	s2 := server.NewServer(server.Port(54322), server.Name("a.b.c"), server.Tracer(tracer))
	s2.GET("/root/aa", func(c *server.Context) {
		c.Response.WriteString("hello root aa")
	})
	// start
	go func() {
		err := s2.Run()
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(1 * time.Second)

	nc := NewClient(Tracer(tracer))
	req := NewRequest(context.Background()).WithURL("http://localhost:54321/root/bb")
	req.SetBaggage("1111", "2222")
	nc.Call(req)

	invalidPath1 := "api/payment/wxapp/ or if(now()=sysdate(),sleep(2),0)"
	invalidPath2 := "api/payment/wxapp/'\"\\'\\\");|]*{\r\n\u003c\u003e\ufffd'"
	_ = invalidPath1
	_ = invalidPath2
	req2 := NewRequest(context.Background()).WithURL("http://localhost:54321/" + invalidPath2)
	_, err = nc.Call(req2)
	if err != nil {
		fmt.Println(err)
	}

	s.Stop()
}

func TestRetryTimes(t *testing.T) {
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
	s := server.NewServer(server.Port(27235), server.Name("a.b.c"), server.Tracer(tracer))
	// register
	s.POST("/jake", func(c *server.Context) {
		time.Sleep(time.Second)
		c.Response.WriteHeader(200)
		c.Response.WriteString("{\"text\":\"hello v1\"}")
		c.Response.Write([]byte(`\n1111111`))
	})
	// start
	go func() {
		err := s.Run()
		if err != nil {
			panic(err)
		}
	}()
	time.Sleep(1 * time.Second)

	// cluster
	clusterName := "test_client"
	config := config.NewCluster()
	config.Name = clusterName
	config.StaticEndpoints = "localhost:27235"
	manager := upstream.NewClusterManager()
	manager.InitService(config)

	// val := "b"

	nc2 := NewClient(Tracer(tracer), Cluster(manager.Cluster(clusterName)), ProtoType("http"))
	ro := &RequestOption{}
	ro.RetryTimes(3).RequestTimeoutMS(200)
	req2 := NewRequest(context.Background()).
		WithOption(ro).
		// WithURL(fmt.Sprintf("/v1?a=%s", val)).
		WithPath("/:name").
		WithPathParams(map[string]string{"name": "jake"}).
		WithBody(nil).
		WithMethod("POST").
		WithQueryParam("a", "b").WithBody(bytes.NewBuffer([]byte(`1234567890`)))

	req2.RawRequest()
	rsp2, err := nc2.Call(req2)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%+v\n", rsp2.String())
	}

	time.Sleep(1 * time.Second)
	s.Stop()
}

func TestRequest_WithURL(t *testing.T) {

	// 	values, _ := url.ParseRequestURI("https://www.baidu.com/s?wd=%E6%90%9C%E7%B4%A2&rsv_spt=1&issp=1&f=8&rsv_bp=0&rsv_idx=2&ie=utf-8&tn=baiduhome_pg&rsv_enter=1&rsv_sug3=7&rsv_sug1=6")
	//
	// 	fmt.Println(values)
	//
	// 	urlParam := values.RawQuery
	// fmt.Println(urlParam)
	//
	// 	fmt.Println(url.ParseQuery(urlParam))
	//

	nspval := url.Values{}
	nspCtx := fmt.Sprintf("{\"ver\":\"1\", \"appId\":\"%s\"}", "10000")
	nspval.Add("nsp_ctx", nspCtx)

	uri := fmt.Sprintf("http://a.b.c/%s?%s&a=b", "push/send/:aa", nspval.Encode())

	req := NewRequest(context.Background()).WithURL(uri).WithPathParams(map[string]string{"aa": "bb"}).WithFormData(map[string]string{"1": "2"})
	req.build()
	fmt.Printf(">>>1, %+v\n", *req.RawRequest().URL)

	u, err := url.ParseRequestURI(uri)
	if err != nil {
		panic(fmt.Errorf("parse uri failed"))
	}
	fmt.Printf(">>>2, %+v\n", *u)

	// req.parseURL()

	fmt.Printf(">>>3, %+v\n", *req.RawRequest().URL)

}

func TestRegisterOnGlobalStage(t *testing.T) {
	// server
	s := server.NewServer(server.Port(42345), server.Name("a.b.c"))

	// register
	s.GET("/v2", func(c *server.Context) {
		c.Response.WriteString("hello world")
	})

	server.RegisterOnGlobalStage(func(c *server.Context) {
		fmt.Printf("\non server global....\n")
	})

	server.RegisterOnRequestStage(func(c *server.Context) {
		fmt.Printf("\non server request....\n")
	})

	// start
	go func() {
		err := s.Run()
		if err != nil {
			panic(err)
		}
	}()
	time.Sleep(1 * time.Second)

	nc2 := NewClient()
	req2 := NewRequest(context.Background()).
		WithURL("http://localhost:42345/v2").
		WithBody(bytes.NewBuffer([]byte(`hello world`)))

	RegisterOnGlobalStage(func(c *Context) {
		fmt.Println("on client global....")
	}, func(c *Context) {

	})

	RegisterOnRequestStage(func(c *Context) {
		fmt.Println("on client request....")
	})

	rsp2, err := nc2.Call(req2)
	if err != nil {
		panic(err)
	}
	fmt.Println(rsp2)
	s.Stop()
}

func TestNewClient(t *testing.T) {
	//
	// // init tracer
	// cfg := jaegerconfig.Configuration{
	// 	// SamplingServerURL: "http://localhost:5778/sampling"
	// 	Sampler: &jaegerconfig.SamplerConfig{Type: jaeger.SamplerTypeRemote},
	// 	Reporter: &jaegerconfig.ReporterConfig{
	// 		LogSpans:            false,
	// 		BufferFlushInterval: 1 * time.Second,
	// 		LocalAgentHostPort:  "127.0.0.1:6831",
	// 	},
	// }
	// tracer, _, err := cfg.New("abc")
	// if err != nil {
	// 	panic(err)
	// }
	//
	// client := NewClient(Tracer(tracer))
	// span := tracer.StartSpan("HTTP Client GET /ping")
	// span.SetTag("aa", "bb")
	// fmt.Println(span)
	// ctx := opentracing.ContextWithSpan(context.Background(), span)
	// ro := &RequestOption{}
	// ro.ServiceName("11.22.33")
	// ro.CallerName("aaa.bbb.ccc")
	// request := NewRequest().WithURL("http://10.55.3.187:1234/ping").WithOption(ro).WithCtxInfo(ctx)
	// resp, err := client.Call(request)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(resp.String())

	l, _ := logging.NewJSON("/tmp/aa.txt", rolling.DailyRolling)
	logItems := []interface{}{
		"type", "httpclient",
		"cost", "1",
		"trace_id", "aaa",
		"service_name", "xxx",
		"req_method", "a1",
		"req_uri", "a2",
		"http_code", "200",
		"address", "host",
	}
	l.Debugw("a_b", logItems...)

	l.Sync()
}

func TestRequest_WithBigBody(t *testing.T) {
	fd, err := os.Open("/tmp/signal.log")
	if err != nil {
		return
	}
	defer fd.Close()

	s := server.NewServer(server.Port(26767), server.Name("a.b.c"), server.RespBodyLogMaxSize(0))
	s.POST("/abc", func(c *server.Context) {
		b, _ := ioutil.ReadAll(c.Request.Body)
		fmt.Println(">>>>> req body", len(b))
		// time.Sleep(1 * time.Second)
		fd2, err := os.Open("/tmp/signal.log.1")
		if err != nil {
			c.Response.WriteHeader(500)
			return
		}
		n, err := io.Copy(c.Response.Writer(), fd2)
		if err != nil {
			fmt.Println("copy err:", err)
		}
		fmt.Println("copy size:", n)

		fd2.Close()
		// c.Response.WriteString("hello world")
		fmt.Println("on server:", c.Response.Size())
	})

	go func() {
		err := s.Run()
		if err != nil {
			panic(err)
		}
	}()
	time.Sleep(1 * time.Second)

	// fd := bytes.NewBuffer([]byte(`123456`))
	req := NewRequest(context.Background()).WithMethod("POST").WithURL("http://localhost:26767/abc").WithBigBody(fd)

	req.RawRequest()

	nc := NewClient(RetryTimes(3), RequestTimeout(100*time.Millisecond))
	rsp, err := nc.Call(req)
	if err != nil {
		panic(err)
	}
	fmt.Println(rsp.Code())
	fmt.Println(len(rsp.Bytes()))
}

func TestRetryDataRace(t *testing.T) {
	s := server.NewServer(server.Port(41145), server.Name("a.b.c"))
	s.GET("/data_race", func(c *server.Context) {
		time.Sleep(1 * time.Second)
		c.JSON(nil, nil)
	})
	go func() {
		err := s.Run()
		if err != nil {
			panic(err)
		}
	}()
	time.Sleep(1 * time.Second)

	// use the same client
	nc := NewClient(RetryTimes(3), RequestTimeout(100*time.Millisecond))
	var wg sync.WaitGroup
	wg.Add(50)
	for i := 0; i < 50; i++ {
		go func() {
			defer wg.Done()
			req := NewRequest(context.Background()).WithMethod("GET").WithURL("http://localhost:41145/data_race")
			rsp, err := nc.Call(req)
			if err != nil {
				fmt.Println("error:", err)
				return
			}
			fmt.Println("result:", rsp.String())
		}()
	}
	wg.Wait()
}
