package core

import "context"

// Core 插件管理器接口
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
	// Err 获取执行中的错误
	Err() error
	// Copy 复制
	Copy() Core
	// Index 索引
	Index() int
	// Reset 重置
	Reset(idx int)
}
