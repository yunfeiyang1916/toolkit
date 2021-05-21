package sd

import (
	"fmt"
	"strings"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/namespace"
	"github.com/yunfeiyang1916/toolkit/go-upstream/upstream"
	"github.com/yunfeiyang1916/toolkit/logging"
	"golang.org/x/net/context"
)

type ikconsul struct {
	f       Factory
	cluster *upstream.Cluster
}

func Upstream(factory Factory, cluster *upstream.Cluster) core.Plugin {
	return ikconsul{
		cluster: cluster,
		f:       factory,
	}
}

const (
	resultSuccess upstream.Result = 0
	// resultTimeout                        = 1
	// resultTemporary                      = 2
	resultDNS = 3
	// resultRefused                        = 4
	resultConnectTimeout = 5
	// resultSysCall                        = 6
	resultEOF = 7
	// resultConnectUnknown                 = 99
	resultDeadline = 103
	// resultRPCUser                        = 1000
	// resultRPCUnkown                      = 1001
	resultCallError = 199
	// resultUnknown                        = 200
)

func getResultFromError(err error) upstream.Result {
	switch err := err.(type) {
	case nil:
		return resultSuccess
	default:
		if strings.Contains(err.Error(), "lookup") {
			return resultDNS
		}
		if strings.Contains(err.Error(), "canceld") {
			return resultSuccess
		}
		if strings.Contains(err.Error(), "exceed") {
			return resultDeadline
		}
		if strings.Contains(err.Error(), "timeout") {
			return resultConnectTimeout
		}
		if strings.Contains(err.Error(), "EOF") {
			return resultEOF
		}
		switch err {
		case context.DeadlineExceeded:
			return resultDeadline
		case context.Canceled:
			return resultSuccess
		default:
			return resultCallError
		}
	}
}

const (
	EnvKey      = "env"
	pressureEnv = "loadtest"
)

func matched(reqEnv string, reqApp string, m map[string]string) (envOk, appOk bool) {
	if m == nil {
		return
	}
	if v, ok := m[EnvKey]; ok && len(reqEnv) > 0 && reqEnv == v {
		envOk = true
	}
	if v, ok := m[namespace.NAMESPACE]; ok && len(reqApp) > 0 && reqApp == v {
		appOk = true
	}
	return
}

func chooseHost(b upstream.Balancer, ctx context.Context) (host *upstream.Host, err error) {
	envPair := make([]string, 0, 2)
	appPair := make([]string, 0, 2)
	var reqEnv string
	var reqApp string
	nCtx := ctx
	// lb_default_keys default is "online", on init stage
	if span := opentracing.SpanFromContext(ctx); span != nil {
		reqEnv = span.BaggageItem(EnvKey)
		reqApp = span.BaggageItem(namespace.NAMESPACE)
		if len(reqEnv) > 0 {
			envPair = []string{EnvKey, reqEnv}
			if len(reqApp) > 0 {
				appPair = []string{namespace.NAMESPACE, reqApp}
			}
			list := make([]string, 0, 4)
			list = append(list, envPair...)
			list = append(list, appPair...)
			nCtx = upstream.InjectSubsetCarrier(ctx, list)
		}
	}

	check := false
Again:
	host = b.ChooseHost(nCtx)
	if host == nil { // 如果默认策略配置了,但是无法找到相应host,则中断执行
		return nil, fmt.Errorf("no living upstream")
	}

	// 1.流量标记了,也命中相应的标记服务,则流量路由过去.
	// 2.流量标记了,但是找不到相应的标记服务,则会命中lb_default_keys策略,选online机器.
	// 3.流量没有被标记,则应该都被路由到online.
	// 4.以上情况都无法命中,即无法找到默认online环境的目标服务机器,则拒绝请求.
	envOk, appOk := matched(reqEnv, reqApp, host.Meta())

	if envOk && appOk { // 完全命中
		return
	}
	if reqEnv == pressureEnv && !envOk { // 没找到压测标记的host, 直接中断执行
		logging.GenLogf("on upstream, env %s host not found, break off!", reqEnv)
		return nil, fmt.Errorf("no living upstream, on env %s", reqEnv)
	}
	if !check {
		check = true
		// 在env+app方式无法命中的情况下会走默认策略,此时在env和app都存在的情况下需要重试,并只选择env配置的机器
		if !envOk && len(envPair) > 0 && len(appPair) > 0 {
			logging.GenLogf("on upstream, not hit env and app, host: %s, %+v, inject env %v try again.", host.Address(), host.Meta(), envPair)
			nCtx = upstream.InjectSubsetCarrier(ctx, envPair)
			goto Again
		}
	}
	// 1.命中env返回
	// 2.使用默认策略
	return
}

func (up ikconsul) Do(ctx context.Context, c core.Core) {
	if up.cluster == nil {
		return
	}
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span.LogFields(opentracinglog.String("event", "discover upstream service"))
	}
	host, err := chooseHost(up.cluster.Balancer(), ctx)
	if err != nil {
		c.AbortErr(err)
		return
	}
	plugin, err := up.f.Factory(host.Address())
	host.GetDetectorMonitor().PutResult(getResultFromError(err))
	if err != nil {
		c.AbortErr(err)
		if span != nil {
			span.LogFields(opentracinglog.String("event", "error"), opentracinglog.Error(err))
			ext.Error.Set(span, true)
		}
		return
	}
	plugin.Do(ctx, c)
}
