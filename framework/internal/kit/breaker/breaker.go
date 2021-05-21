package breaker

import (
	"github.com/yunfeiyang1916/toolkit/framework/breaker"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"golang.org/x/net/context"
)

type ikBreaker struct {
	namespace  string
	clientname string
	name       string
	brkConfig  *breaker.Config
}

// TODO failback
func Breaker(namespace, clientname, name string, brk *breaker.Config) core.Plugin {
	return ikBreaker{namespace, clientname, name, brk}
}

func (i ikBreaker) Do(ctx context.Context, flow core.Core) {
	if i.brkConfig == nil {
		return
	}

	brk := i.brkConfig.GetBreaker(breaker.ClientBreakerType, i.namespace, i.clientname, i.name)
	if brk == nil {
		return
	}

	if err := brk.Call(func() error {
		flow.Next(ctx)
		return flow.Err()
	}, 0); err != nil {
		flow.AbortErr(err)
	}
}
