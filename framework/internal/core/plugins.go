package core

import "context"

// Plugin 插件接口
type Plugin interface {
	// Do 处理
	Do(context.Context, Core)
}

type Function func(context.Context, Core)

func (f Function) Do(ctx context.Context, core Core) {
	f(ctx, core)
}
