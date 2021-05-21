package ikiosocket

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yunfeiyang1916/toolkit/framework/log"
)

func TestServer(t *testing.T) {
	assert := assert.New(t)
	host := "0.0.0.0:12345"
	ln, err := net.Listen("tcp4", host)
	assert.Nil(err)

	cb := func(addr string, request *Context) (*Context, error) {
		return &Context{
			Body: append(request.Body, byte('#')),
			Header: map[string]string{
				"body": string(request.Body),
				"i":    request.Header["i"],
			},
		}, nil
	}

	s := NewServer(log.Stdout(), cb)
	go func() {
		assert.Nil(s.Start(ln))
	}()

	c := New(log.Stdout())
	assert.Nil(c.Start(host, time.Second))

	wg := &sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		i := i + 1
		wg.Add(1)
		go func() {
			defer wg.Done()
			request := &Context{
				Header: map[string]string{
					"i": fmt.Sprintf("%d", i),
				},
				Body: []byte(fmt.Sprintf("%d", i)),
			}
			response, err := c.Call(request, Timeout(time.Second*30))
			if assert.Nil(err) {
				assert.Equal(response.Body, append(request.Body, byte('#')))
				assert.EqualValues(request.Body, response.Header["body"])
				assert.EqualValues(request.Header["i"], response.Header["i"])
			}
		}()
	}
	wg.Wait()
}
