package server

import (
	"fmt"

	"github.com/yunfeiyang1916/toolkit/framework/breaker"

	"github.com/pkg/errors"

	"github.com/yunfeiyang1916/toolkit/framework/ratelimit"
)

func BreakerPlugin(c *Context) {
	if c.opts.Breaker == nil {
		return
	}

	endpoint := fmt.Sprintf("%s.%s", c.Service, c.Method)
	brk := c.opts.Breaker.GetBreaker(breaker.ServerBreakerType, c.Namespace, "", endpoint)
	if brk == nil {
		return
	}

	err := brk.Call(func() error {
		c.Next()
		return c.Err()
	}, 0)
	if err != nil {
		c.AbortErr(err)
	}
}

func RatelimitPlugin(c *Context) {
	if c.opts.Limiter == nil {
		return
	}
	endpoint := fmt.Sprintf("%s.%s", c.Service, c.Method)
	lim := c.opts.Limiter.GetLimiter(ratelimit.ServerLimiterType, c.Namespace, c.Peer, endpoint)
	if lim != nil && !lim.Allow() {
		err := c.Err()
		if err != nil {
			err = errors.Wrap(err, ratelimit.ErrLimited.Error())
		} else {
			err = ratelimit.ErrLimited
		}
		c.AbortErr(err)
	}
}
