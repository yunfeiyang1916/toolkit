package third

import (
	"sync"

	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
)

type Third struct {
	g *globalStage
	r *requestStage
}

func New() *Third {
	return &Third{
		g: &globalStage{
			ps: make([]core.Plugin, 0),
		},

		r: &requestStage{
			ps: make([]core.Plugin, 0),
		},
	}
}

func (t *Third) OnGlobalStage() Middleware {
	return t.g
}

func (t *Third) OnRequestStage() Middleware {
	return t.r
}

type Middleware interface {
	Register([]core.Plugin)
	Stream() []core.Plugin
}

type globalStage struct {
	mu sync.Mutex
	ps []core.Plugin
}

func (g *globalStage) Register(ps []core.Plugin) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ps = append(g.ps, ps...)
}

func (g *globalStage) Stream() []core.Plugin {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.ps
}

type requestStage struct {
	mu sync.Mutex
	ps []core.Plugin
}

func (r *requestStage) Register(ps []core.Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ps = append(r.ps, ps...)
}

func (r *requestStage) Stream() []core.Plugin {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ps
}
