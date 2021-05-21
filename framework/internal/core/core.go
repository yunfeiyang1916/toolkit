package core

import "context"

// Core 核心接口
type Core interface {
	// Use 应用插件
	Use(...Plugin) Core
	// Next 下一步
	Next(context.Context)
	// AbortErr 放弃后续执行并设置错误
	AbortErr(error)
	// Abort 放弃后续执行
	Abort()
	// IsAborted 是否已放弃
	IsAborted() bool
	Err() error
	// Copy 复制
	Copy() Core
	Index() int
	Reset(idx int)
}
