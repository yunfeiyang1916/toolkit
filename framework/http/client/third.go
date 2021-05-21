package client

import (
	"github.com/yunfeiyang1916/toolkit/framework/http/third"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
)

var clientInternalThirdPlugin = third.New()

// 需要在最开始的时候注册插件

// plugins will be effect always
func RegisterOnGlobalStage(plugFunc ...HandlerFunc) {
	ps := make([]core.Plugin, len(plugFunc))
	for i := range plugFunc {
		ps[i] = plugFunc[i]
	}
	clientInternalThirdPlugin.OnGlobalStage().Register(ps)
}

// plugins will be effect for a http request or a http route
func RegisterOnRequestStage(plugFunc ...HandlerFunc) {
	ps := make([]core.Plugin, len(plugFunc))
	for i := range plugFunc {
		ps[i] = plugFunc[i]
	}
	clientInternalThirdPlugin.OnRequestStage().Register(ps)
}
