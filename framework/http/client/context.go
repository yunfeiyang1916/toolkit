package client

import (
	"context"
	"net/http"

	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
)

type internalContext struct{}

var iCtxKey = internalContext{}

type internalReqPath struct{}

var iReqPathKey = internalReqPath{}

type Context struct {
	Ctx           context.Context
	Req           *Request
	Resp          *Response
	core          core.Core
	orgReq        *http.Request
	orgCtx        context.Context
	simpleBaggage map[string]string
}

func newContext(req *Request) *Context {
	c := &Context{
		Ctx:    req.ctx,
		Req:    req,
		Resp:   nil,
		core:   nil,
		orgCtx: req.ctx,
	}
	// 在call之前build出request,需要缓存下该原始request,重试时候使用
	if c.Req.raw != nil {
		c.orgReq = cloneRawRequest(c.orgCtx, c.Req.raw)
	}
	return c
}

func (c *Context) Next() {
	c.core.Next(c.Ctx)
}

func (c *Context) Abort() {
	c.core.Abort()
}

func (c *Context) AbortErr(err error) {
	c.core.AbortErr(err)
}

func (c *Context) Err() error {
	return c.core.Err()
}
