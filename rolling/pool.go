package rolling

import (
	"bytes"
	"sync"
	"sync/atomic"
)

var bpool = newBufferPool(500)

type bufferPool struct {
	p     sync.Pool
	count int64
	size  int
}

func newBufferPool(size int) *bufferPool {
	p := &bufferPool{}
	p.p.New = func() interface{} { return bytes.NewBuffer(nil) }
	p.size = size
	return p
}

func putBuffer(b *bytes.Buffer) {
	bpool.p.Put(b)
	atomic.AddInt64(&bpool.count, -1)
}

func getBuffer() *bytes.Buffer {
	for {
		c := atomic.LoadInt64(&bpool.count)
		if c > int64(bpool.size) {
			return nil
		}
		if atomic.CompareAndSwapInt64(&bpool.count, c, c+1) {
			b := bpool.p.Get().(*bytes.Buffer)
			b.Reset()
			return b
		}
	}
}
