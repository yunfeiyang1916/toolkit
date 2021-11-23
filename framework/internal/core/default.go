package core

import "context"

// defaultCore 插件管理器默认实现
type defaultCore struct {
	// 插件集合
	plugins []Plugin
	// 索引，表示执行到第几个插件了
	index int
	err   error
}

// New 构造插件处理器
func New(ps []Plugin) Core {
	return &defaultCore{
		plugins: ps,
		index:   -1,
		err:     nil,
	}
}

// Use 应用插件
func (c *defaultCore) Use(ps ...Plugin) Core {
	c.plugins = append(c.plugins, ps...)
	return c
}

// Next 下一步
func (c *defaultCore) Next(ctx context.Context) {
	c.index++
	for s := len(c.plugins); c.index < s; c.index++ {
		c.plugins[c.index].Do(ctx, c)
	}
}

// Abort 放弃后续执行
func (c *defaultCore) Abort() {
	c.index = len(c.plugins)
}

// AbortErr 放弃后续执行并设置错误
func (c *defaultCore) AbortErr(err error) {
	c.Abort()
	c.err = err
}

// Err 获取执行中的错误
func (c *defaultCore) Err() error {
	return c.err
}

// IsAborted 是否已放弃
func (c *defaultCore) IsAborted() bool {
	return c.index >= len(c.plugins)
}

// Index 索引
func (c *defaultCore) Index() int {
	return c.index
}

func (c *defaultCore) Reset(n int) {
	c.index = n
	c.err = nil
}

// Deprecated: Copy, just use Index()
func (c *defaultCore) Copy() Core {
	dup := &defaultCore{}
	dup.index = c.index
	dup.plugins = append(c.plugins[:0:0], c.plugins...)
	return dup
}
