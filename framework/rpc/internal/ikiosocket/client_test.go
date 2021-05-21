package ikiosocket

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yunfeiyang1916/toolkit/framework/log"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/internal/ikiosocket/mocks"
	"github.com/yunfeiyang1916/toolkit/ikio"
	"golang.org/x/net/context"
)

// testable WriteCloser
type writeCloser struct {
	mocks.WriteCloser

	socket *IKIOSocket
	nego   int64
	assert *assert.Assertions
}

func newMockWC(assert *assert.Assertions, socket *IKIOSocket) *writeCloser {
	wc := &writeCloser{
		socket: socket,
		assert: assert,
	}
	wc.On("Write", mock.Anything, mock.Anything).Return(0, wc.mockResultFn)
	return wc
}

func (wc *writeCloser) mockResultFn(_ context.Context, pkt ikio.Packet) error {
	switch pkt := pkt.(type) {
	case *RPCNegoPacket:
		wc.assert.True(atomic.CompareAndSwapInt64(&wc.nego, 0, 1))
	case *RPCPacket:
		// mush have nego
		wc.assert.EqualValues(1, atomic.LoadInt64(&wc.nego))
		pkt.Tp = PacketTypeResponse
		if _, ok := pkt.GetHeader([]byte("timeout")); !ok {
			// just echo
			wc.socket.onMessage(pkt, wc)
		}
	default:
		wc.assert.Fail("should't be here")
	}
	return nil
}

func (wc *writeCloser) onConnect() bool {
	return wc.socket.onConnect(wc)
}

func (wc *writeCloser) onClose() {
	wc.On("Close").Return(nil)
	wc.socket.onClose(wc)
}

func TestSocketGeneral(t *testing.T) {
	socket := New(log.Stdout())
	wc := newMockWC(assert.New(t), socket)

	// test onConnect
	assert.True(t, wc.onConnect())

	// after connected, wc.nego must bu 1
	assert.EqualValues(t, 1, wc.nego)

	// start
	socket.startWC(wc)

	tests := map[string]struct {
		reqContext *Context
		context    *Context
		err        error
	}{
		"nil": {
			reqContext: &Context{nil, nil},
			context:    &Context{make(map[string]string), nil},
			err:        nil,
		},
		"normal": {
			reqContext: &Context{nil, []byte("body")},
			context:    &Context{make(map[string]string), []byte("body")},
			err:        nil,
		},
		"timeout": {
			reqContext: &Context{
				Header: map[string]string{
					"timeout": "",
				},
				Body: []byte("body"),
			},
			context: nil,
			err:     ErrTimeout,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			response, err := socket.Call(test.reqContext)
			assert.Equal(t, test.context, response)
			assert.Equal(t, test.err, err)
		})
	}
}

func TestSocketClose(t *testing.T) {
	socket := New(log.Stdout())
	wc := newMockWC(assert.New(t), socket)

	// test onConnect
	assert.True(t, wc.onConnect())

	// after connected, wc.nego must bu 1
	assert.EqualValues(t, 1, wc.nego)

	// start
	socket.startWC(wc)

	n := 300
	wait := make(chan struct{})
	go func() {
		<-wait
		wc.onClose()
	}()

	wg := &sync.WaitGroup{}
	for i := 0; i < n; i++ {
		reqContext := &Context{
			Header: map[string]string{
				"timeout": "",
			},
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			response, err := socket.Call(reqContext, Timeout(time.Second*1))
			assert.Contains(t, []error{ErrExited, ErrTimeout}, err)
			assert.Nil(t, response)
		}()
	}
	close(wait)
	wg.Wait()
}

func TestSocketCall(t *testing.T) {
	socket := New(log.Stdout())
	wc := newMockWC(assert.New(t), socket)

	// test onConnect
	assert.True(t, wc.onConnect())

	// after connected, wc.nego must be 1
	assert.EqualValues(t, 1, wc.nego)

	// start
	socket.startWC(wc)

	n := 1000
	wg := &sync.WaitGroup{}
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			reqContext := &Context{
				Header: map[string]string{
					"call": fmt.Sprintf("%d", i),
				},
				Body: []byte(fmt.Sprintf("%d", i)),
			}

			response, err := socket.Call(reqContext, Timeout(time.Second*20))
			assert.Nil(t, err)
			assert.Equal(t, reqContext, response)
		}()
	}
	wg.Wait()
}
