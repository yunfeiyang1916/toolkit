package sd

//import (
//	"context"
//	"fmt"
//	"strings"
//	"sync"
//	"testing"
//	"time"
//
//	"git.inke.cn/BackendPlatform/jaeger-client-go"
//	jaegerconfig "git.inke.cn/BackendPlatform/jaeger-client-go/config"
//	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/namespace"
//	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
//	"github.com/yunfeiyang1916/toolkit/go-upstream/upstream"
//	"github.com/opentracing/opentracing-go"
//	"github.com/stretchr/testify/assert"
//)
//
//func TestChooseHost(t *testing.T) {
//	// init tracer
//	cfg := jaegerconfig.Configuration{
//		// SamplingServerURL: "http://localhost:5778/sampling"
//		Sampler: &jaegerconfig.SamplerConfig{Type: jaeger.SamplerTypeRemote},
//		Reporter: &jaegerconfig.ReporterConfig{
//			LogSpans:            false,
//			BufferFlushInterval: 1 * time.Second,
//			LocalAgentHostPort:  "127.0.0.1:6831",
//		},
//	}
//	tracer, _, err := cfg.New("abc")
//	if err != nil {
//		panic(err)
//	}
//
//	// cluster config
//	conf := config.NewCluster()
//	conf.Name = "upstream_test"
//	conf.LBType = "WeightRoundRobin"
//	conf.Proto = "http"
//	// 关心两种策略:env 或 env+app
//	conf.LBSubsetKeys = append(conf.LBSubsetKeys, []string{EnvKey}, []string{EnvKey, namespace.NAMESPACE})
//	// conf.EndpointsFrom = "consul"
//
//	// 默认策略
//	conf.LBDefaultKeys = []string{"env", "online"}
//
//	balanceType := upstream.UnmarshalBalanceFromText(conf.LBType)
//	h0 := upstream.NewHost("8.8.8.8:8888", 100, map[string]string{})
//	h1 := upstream.NewHost("1.1.1.1:1111", 100, map[string]string{"env": ""})
//
//	h2 := upstream.NewHost("2.2.2.2:2222", 100, map[string]string{"env": "online"})
//	h6 := upstream.NewHost("6.6.6.6:6666", 100, map[string]string{"env": "online"})
//	h7 := upstream.NewHost("7.7.7.7:7777", 100, map[string]string{"env": "online"})
//
//	h3 := upstream.NewHost("3.3.3.3:3333", 100, map[string]string{"env": "preview"})
//	h4 := upstream.NewHost("4.4.4.4:4444", 100, map[string]string{"env": "loadtest"})
//	h5 := upstream.NewHost("5.5.5.5:5555", 100, map[string]string{"env": "xxxx"})
//
//	h8 := upstream.NewHost("9.9.9.9:9999", 100, map[string]string{"env": "eeee", namespace.NAMESPACE: "good"})
//	// h8 := upstream.NewHost("9.9.9.9:9999", 100, map[string]string{"env": "eeee"})
//	h9 := upstream.NewHost("1.2.3.4:1234", 100, map[string]string{"env": "eeee"})
//
//	hlist := []*upstream.Host{h0, h1, h2, h3, h4, h5, h6, h7, h8, h9}
//
//	// hlist := []*upstream.Host{h0, h1, h3, h4, h5, h8, h9}
//
//	hostSet := upstream.NewHostSet(hlist, hlist)
//	b := upstream.NewSubsetBalancer(hostSet, balanceType, conf.LBPanicThreshold, conf.LBSubsetKeys, conf.LBDefaultKeys, conf.Name)
//	loop(t, b, tracer)
//}
//
//func makeCtx(opname, env string, tracer opentracing.Tracer) context.Context {
//	span := tracer.StartSpan(opname)
//	if env != "null" {
//		span.SetBaggageItem(EnvKey, env)
//	}
//	if env == "eeee" {
//		span.SetBaggageItem(namespace.NAMESPACE, "good")
//	}
//	ctx := opentracing.ContextWithSpan(context.Background(), span)
//	// span2 := opentracing.SpanFromContext(ctx)
//	// vv := span2.BaggageItem(EnvKey)
//	// fmt.Println(vv)
//	return ctx
//}
//
//type data struct {
//	ctx context.Context
//	env string
//}
//
//func loop(t *testing.T, b upstream.Balancer, tracer opentracing.Tracer) {
//	fmt.Printf("*******************\n\n")
//	dataList := []data{
//		{ctx: makeCtx("test1", "", tracer), env: "online"},       // online
//		{ctx: makeCtx("test1", "null", tracer), env: "online"},   // online
//		{ctx: makeCtx("test2", "xxxx", tracer), env: "xxxx"},     // xxxx
//		{ctx: makeCtx("test3", "online", tracer), env: "online"}, // online
//
//		{ctx: makeCtx("test4", "preview", tracer), env: "preview"}, // preview
//
//		{ctx: makeCtx("test5", "loadtest", tracer), env: "loadtest"}, // loadtest or break off
//
//		{ctx: makeCtx("test6", "aaaa", tracer), env: "online"}, // online
//		{ctx: makeCtx("test7", "bbbb", tracer), env: "online"}, // online
//		{ctx: makeCtx("test8", "cccc", tracer), env: "online"}, // online
//		{ctx: makeCtx("test9", "dddd", tracer), env: "online"}, // online
//
//		{ctx: makeCtx("test10", "eeee", tracer), env: "eeee"}, // eeee
//		{ctx: makeCtx("test10", "eeee", tracer), env: "eeee"}, // eeee
//		{ctx: makeCtx("test10", "eeee", tracer), env: "eeee"}, // eeee
//	}
//
//	serial(t, b, dataList)
//
//	time.Sleep(1 * time.Second)
//	fmt.Printf("\n\n=============================\n\n")
//
//	parallel(t, b, dataList)
//
//}
//
//func serial(t *testing.T, b upstream.Balancer, dataList []data) {
//	for _, c := range dataList {
//		d := c
//		host, err := chooseHost(b, d.ctx)
//		if err != nil {
//			if strings.Contains(err.Error(), d.env) {
//				// continue
//				return
//			}
//			panic(err)
//		}
//		env := host.Meta()["env"]
//		_ = env
//		assert.Equal(t, env, d.env)
//		fmt.Printf(">>>> on upstream, use host: %s, %+v\n", host.Address(), host.Meta())
//	}
//}
//
//func parallel(t *testing.T, b upstream.Balancer, dataList []data) {
//	wg := sync.WaitGroup{}
//	wg.Add(len(dataList))
//	for _, c := range dataList {
//		go func(d data) {
//			defer wg.Done()
//			host, err := chooseHost(b, d.ctx)
//			if err != nil {
//				if strings.Contains(err.Error(), d.env) {
//					// continue
//					return
//				}
//				panic(err)
//			}
//			env := host.Meta()["env"]
//			_ = env
//			assert.Equal(t, env, d.env)
//			fmt.Printf(">>>> on upstream, use host: %s, %+v\n", host.Address(), host.Meta())
//		}(c)
//	}
//	wg.Wait()
//}
