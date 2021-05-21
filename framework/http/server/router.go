package server

import (
	"path"
	"sync"

	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
)

// router
type Router interface {
	GROUP(string, ...HandlerFunc) *RouterMgr
	ANY(string, ...HandlerFunc) Router
	GET(string, ...HandlerFunc) Router
	POST(string, ...HandlerFunc) Router
	GETPOST(string, ...HandlerFunc) Router
	DELETE(string, ...HandlerFunc) Router
	PATCH(string, ...HandlerFunc) Router
	PUT(string, ...HandlerFunc) Router
	OPTIONS(string, ...HandlerFunc) Router
	HEAD(string, ...HandlerFunc) Router
}

type RouterMgr struct {
	plugins  []core.Plugin
	basePath string
	server   *server
	mu       sync.Mutex
}

// 实现Router
func (mgr *RouterMgr) GROUP(relativePath string, handleFunc ...HandlerFunc) *RouterMgr {
	ps := make([]core.Plugin, len(handleFunc))
	for i, h := range handleFunc {
		ps[i] = h
	}
	return &RouterMgr{
		plugins:  mgr.combineHandlers(ps),
		basePath: mgr.absolutePath(relativePath),
		server:   mgr.server,
		mu:       sync.Mutex{},
	}
}

func (mgr *RouterMgr) GETPOST(relativePath string, handlers ...HandlerFunc) Router {
	mgr.handle("GET", relativePath, handlers...)
	mgr.handle("POST", relativePath, handlers...)
	return mgr
}

func (mgr *RouterMgr) POST(relativePath string, handlers ...HandlerFunc) Router {
	return mgr.handle("POST", relativePath, handlers...)
}

func (mgr *RouterMgr) GET(relativePath string, handlers ...HandlerFunc) Router {
	return mgr.handle("GET", relativePath, handlers...)
}

func (mgr *RouterMgr) DELETE(relativePath string, handlers ...HandlerFunc) Router {
	return mgr.handle("DELETE", relativePath, handlers...)
}

func (mgr *RouterMgr) PATCH(relativePath string, handlers ...HandlerFunc) Router {
	return mgr.handle("PATCH", relativePath, handlers...)
}

func (mgr *RouterMgr) PUT(relativePath string, handlers ...HandlerFunc) Router {
	return mgr.handle("PUT", relativePath, handlers...)
}

func (mgr *RouterMgr) OPTIONS(relativePath string, handlers ...HandlerFunc) Router {
	return mgr.handle("OPTIONS", relativePath, handlers...)
}

func (mgr *RouterMgr) HEAD(relativePath string, handlers ...HandlerFunc) Router {
	return mgr.handle("HEAD", relativePath, handlers...)
}

func (mgr *RouterMgr) ANY(relativePath string, handlers ...HandlerFunc) Router {
	mgr.handle("GET", relativePath, handlers...)
	mgr.handle("POST", relativePath, handlers...)
	mgr.handle("PUT", relativePath, handlers...)
	mgr.handle("PATCH", relativePath, handlers...)
	mgr.handle("HEAD", relativePath, handlers...)
	mgr.handle("OPTIONS", relativePath, handlers...)
	mgr.handle("DELETE", relativePath, handlers...)
	mgr.handle("CONNECT", relativePath, handlers...)
	mgr.handle("TRACE", relativePath, handlers...)
	return mgr
}

// router internal func
func (mgr *RouterMgr) handle(httpMethod, relativePath string, handlers ...HandlerFunc) Router {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	absolutePath := mgr.absolutePath(relativePath)
	hds := make([]core.Plugin, len(handlers))
	for i, h := range handlers {
		hds[i] = h
	}
	h1 := mgr.combineHandlers(hds)
	// server has a tree, to add plugins
	mgr.server.addRoute(httpMethod, absolutePath, h1)
	return mgr
}

func (mgr *RouterMgr) combineHandlers(handlers []core.Plugin) []core.Plugin {
	// group handlers  + handlers
	finalSize := len(mgr.plugins) + len(handlers)
	mergedHandlers := make([]core.Plugin, finalSize)
	copy(mergedHandlers, mgr.plugins)
	copy(mergedHandlers[len(mgr.plugins):], handlers)
	return mergedHandlers
}

func (mgr *RouterMgr) absolutePath(relativePath string) string {
	if relativePath == "" || relativePath == mgr.basePath {
		return mgr.basePath
	}
	// path.join总是会把末尾/去掉
	finalPath := path.Join(mgr.basePath, relativePath)
	if relativePath[len(relativePath)-1] == '/' {
		return finalPath + "/"
	}
	return finalPath
}
