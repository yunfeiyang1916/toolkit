package server

import (
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"golang.org/x/net/context"
)

type Plugin func(c *Context)

type Context struct {
	core       core.Core
	opts       Options
	Ctx        context.Context
	Service    string
	Method     string
	RemoteAddr string
	Namespace  string
	Peer       string // 包含app_name的上游service_name
	Code       int32

	// rpc request raw header
	Header map[string]string

	// rpc request raw body
	Body     []byte
	Request  interface{}
	Response interface{}
}

func (c *Context) Next() {
	c.core.Next(c.Ctx)
}

func (c *Context) AbortErr(err error) {
	c.core.AbortErr(err)
}

func (c *Context) Err() error {
	return c.core.Err()
}
