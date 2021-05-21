package metric

import (
	"time"

	"golang.org/x/net/context"

	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/namespace"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/retry"
	"github.com/yunfeiyang1916/toolkit/metrics"
)

type Code interface {
	Code() int
}

type code struct {
	code int
	err  error
}

func (c code) Code() int {
	return c.code
}

func (c code) Error() string {
	return c.err.Error()
}

func Error(c int, err error) error {
	return code{c, err}
}

// TODO append
func Metric(name string, tags ...interface{}) core.Plugin {
	return core.Function(func(ctx context.Context, c core.Core) {
		start := time.Now()
		defer func() {
			var code = 0
			if c, ok := c.Err().(retry.RetryError); ok {
				if cc, ok := c.Final.(Code); ok {
					code = cc.Code()
				}
			}
			if c, ok := c.Err().(Code); ok {
				code = c.Code()
			}
			if c.Err() != nil && code == 0 {
				code = -1
			}
			if ns := namespace.GetNamespace(ctx); ns != "" {
				metrics.Timer(name, start, append(tags, metrics.TagCode, code, "namespace", ns)...)
			} else {
				metrics.Timer(name, start, append(tags, metrics.TagCode, code)...)
			}
		}()
		c.Next(ctx)
	})
}
