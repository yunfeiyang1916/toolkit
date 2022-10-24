package gin

import "net/http"

type Context struct {
	// http请求
	Request *http.Request
}

// 重置上下文
func (c *Context) reset() {

}
