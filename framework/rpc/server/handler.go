package server

import (
	"reflect"
)

type HandlerOption func(*HandlerOptions)

type HandlerOptions struct {
	HandlerName string
}

func HandlerName(name string) HandlerOption {
	return func(o *HandlerOptions) {
		o.HandlerName = name
	}
}

type Handler interface {
	Name() string
	Handler() interface{}
}

type rpcHandler struct {
	name    string // receiver's type name
	handler interface{}
}

func newRpcHandler(handler interface{}, options ...HandlerOption) Handler {
	opts := HandlerOptions{}

	for _, option := range options {
		option(&opts)
	}

	hdlr := reflect.ValueOf(handler)
	name := reflect.Indirect(hdlr).Type().Name()
	if opts.HandlerName != "" {
		name = opts.HandlerName
	}

	return &rpcHandler{
		name:    name,
		handler: handler,
	}
}

func (h *rpcHandler) Name() string {
	return h.name
}

func (h *rpcHandler) Handler() interface{} {
	return h.handler
}
