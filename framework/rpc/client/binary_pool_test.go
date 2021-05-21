package client

import (
	"errors"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type dialerMock struct {
	mock.Mock
}

func (d *dialerMock) Dial(host string) (socket, error) {
	args := d.Called(host)
	if f, ok := args.Get(0).(func(string) socket); ok {
		return f(host), args.Error(1)
	}
	if args.Get(0) != nil {
		return args.Get(0).(socket), args.Error(1)
	}
	return nil, args.Error(1)
}

type fakeSocket struct {
	closed bool
}

func (f *fakeSocket) Close() error {
	f.closed = true
	return nil
}

func (f *fakeSocket) Call(string, map[string]string, []byte) ([]byte, error) {
	return nil, nil
}

func TestPoolGeneral(t *testing.T) {
	var size = 2
	assert := assert.New(t)

	tests := map[string]struct {
		dialErr  error
		host     string
		respSock interface{}
		err      error
	}{
		"success1": {
			dialErr:  nil,
			err:      nil,
			respSock: new(fakeSocket),
			host:     "0.0.0.0",
		},
		"errors": {
			dialErr:  errors.New("some error"),
			err:      errors.New("some error"),
			respSock: nil,
			host:     "0.0.0.10",
		},
	}

	dialer := new(dialerMock)
	p := newPool(size, dialer, time.Second*2, clockwork.NewRealClock())

	for _, test := range tests {
		dialer.On("Dial", test.host).Return(test.respSock, test.dialErr)
		sock, err := p.getSocket(test.host)
		assert.Equal(test.err, err)
		assert.Equal(test.respSock, sock)
	}
}

func TestPoolCleanUp(t *testing.T) {
	var size = 2
	assert := assert.New(t)

	dialer := new(dialerMock)
	sockets := []socket{}
	dialer.On("Dial", mock.Anything).Return(func(string) socket {
		s := new(fakeSocket)
		sockets = append(sockets, s)
		return s
	}, nil)

	tests := []struct {
		host  string
		times int
		size  int
	}{
		{
			host:  "0.0.0.1",
			times: size - 1,
			size:  size - 1,
		},
		{
			host:  "0.0.0.2",
			times: size * 2,
			size:  size,
		},
	}

	clock := clockwork.NewFakeClockAt(time.Now())
	p := newPool(size, dialer, time.Second*60, clock)
	for _, test := range tests {
		for i := 0; i < test.times; i++ {
			sock, err := p.getSocket(test.host)
			assert.Nil(err)
			assert.NotNil(sock)
		}
		assert.Equal(test.size, len(p.sockets[test.host]))
	}
	clock.Advance(100 * time.Second)

	sock, err := p.getSocket("whatever")
	assert.Nil(err)
	assert.NotNil(sock)

	for _, test := range tests {
		assert.Equal(0, len(p.sockets[test.host]))
	}
}

func TestPoolRelease(t *testing.T) {
	var size = 2
	assert := assert.New(t)

	dialer := new(dialerMock)
	tests := []struct {
		host string
		size int
		err  error
	}{
		{
			host: "0.0.0.1",
			size: 1,
			err:  nil,
		},
		{
			host: "0.0.0.2",
			size: 0,
			err:  errors.New("someerror"),
		},
	}

	p := newPool(size, dialer, time.Second*60, clockwork.NewRealClock())
	for _, test := range tests {
		dialer.On("Dial", mock.Anything).Return(&fakeSocket{}, nil)

		sock, _ := p.getSocket(test.host)
		p.release(test.host, sock, test.err)

		if test.err != nil {
			assert.True(sock.(*fakeSocket).closed)
		}

		assert.Equal(test.size, len(p.sockets[test.host]))
	}
}
