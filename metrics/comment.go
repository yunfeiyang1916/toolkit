package metrics

import (
	"sync"
)

var (
	comments = new(sync.Map)
)

func commentAdd(name string, c interface{}) {
	if m, ok := c.(map[int]string); ok {
		comments.LoadOrStore(name, m)
	}
}

func commentGet(name string) map[int]string {
	c, ok := comments.Load(name)
	if !ok {
		return nil
	}
	if m, ok := c.(map[int]string); ok {
		return m
	}
	return nil
}
