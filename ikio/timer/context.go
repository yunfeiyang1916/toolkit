package timer

import (
	"time"
)

type Context struct {
	values interface{}

	interval time.Duration
	slot     int
	circle   int
}

func (c *Context) Value() interface{} {
	return c.values
}
