package ikio

import (
	"bytes"
	"sync"
)

var Buffer bufferPool

type bufferPool struct {
	buffers sync.Pool
}

func (pool *bufferPool) Get() (buffer *bytes.Buffer) {
	if r := pool.buffers.Get(); r != nil {
		buffer = r.(*bytes.Buffer)
		buffer.Reset()
	} else {
		buffer = bytes.NewBuffer(nil)
	}
	return buffer
}

func (pool *bufferPool) Put(buffer *bytes.Buffer) {
	pool.buffers.Put(buffer)
}
