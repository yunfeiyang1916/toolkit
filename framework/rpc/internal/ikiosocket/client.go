package ikiosocket

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yunfeiyang1916/toolkit/ikio"
	"github.com/yunfeiyang1916/toolkit/logging"
	"golang.org/x/net/context"
)

var ErrTimeout = errors.New("ikioSocket: request timeout")
var ErrExited = errors.New("ikioSocket: connection closed")
var ErrChanSize = errors.New("ikioSocket: excced the request channel size")

type requestContext struct {
	end   int64
	seq   int64
	wg    sync.WaitGroup
	timer *Timer

	// request context
	reqContext *Context

	// response
	context *Context
	err     error
}

type Option func(*Options)

type IKIOSocket struct {
	logger     *logging.Logger
	seq        int64
	closed     int64
	peerAddr   string
	localAddr  string
	controller sync.Map
	wc         ikio.WriteCloser
	wg         sync.WaitGroup

	requestC chan *requestContext
	exitC    chan struct{}
}

func New(l *logging.Logger) *IKIOSocket {
	return &IKIOSocket{
		logger:   l,
		requestC: make(chan *requestContext, 1024),
		exitC:    make(chan struct{}),
	}
}

func (i *IKIOSocket) StartWithConn(conn net.Conn) {
	lg := logging.New()
	lg.SetOutputByName("logs/ikio.log")
	lg.SetLevelByString("error")
	iClient := ikio.NewClientChannel(conn,
		ikio.WithLogger(lg),
		ikio.CustomCodecOption(func() ikio.Codec { return &RPCCodec{} }),
		ikio.OnConnectOption(i.onConnect),
		ikio.OnCloseOption(i.onClose),
		ikio.OnMessageOption(i.onMessage),
		ikio.WriterBufferSizeOption(1024),
	)
	iClient.Start()
	i.wc = iClient
	i.peerAddr = conn.RemoteAddr().String()
	i.localAddr = conn.LocalAddr().String()

	i.wg.Add(1)
	go i.dispatch()
}

func (i *IKIOSocket) Start(host string, dial time.Duration) error {
	conn, err := net.DialTimeout("tcp4", host, dial)
	if err != nil {
		return err
	}

	i.StartWithConn(conn)
	return nil
}

// startWC only for test
//nolint:unused
func (i *IKIOSocket) startWC(wc ikio.WriteCloser) {
	i.wc = wc
	i.wg.Add(1)
	go i.dispatch()
}

type Context struct {
	Header map[string]string
	Body   []byte
}

func (i *IKIOSocket) Call(req *Context, options ...Option) (*Context, error) {

	opts := Options{
		RequestTimeout: time.Second,
	}

	for _, o := range options {
		o(&opts)
	}

	request := &requestContext{}
	request.wg.Add(1)
	request.seq = atomic.AddInt64(&i.seq, 1)
	request.reqContext = req
	request.timer = NewTimer(opts.RequestTimeout, func() {
		i.contextEnd(request, nil, ErrTimeout)
	})
	request.timer.Start()

	select {
	case <-i.exitC:
		return nil, ErrExited
	default:
		select {
		case <-i.exitC:
			return nil, ErrExited
		case i.requestC <- request:
		default:
			return nil, ErrChanSize
		}
	}

	request.wg.Wait()
	return request.context, request.err
}

func (i *IKIOSocket) Close() error {
	if !atomic.CompareAndSwapInt64(&i.closed, 0, 1) {
		// already close
		return nil
	}

	i.logger.Debug(
		"message", "connection closed", "remote", i.peerAddr,
		"local", i.localAddr,
	)

	close(i.exitC)

	// we will not close requestC chan here.
	// close(i.requestC)

	i.wg.Wait()

	i.controller.Range(func(key, value interface{}) bool {
		request := value.(*requestContext)
		i.contextEnd(request, nil, ErrExited)
		return true
	})
	return i.wc.Close()
}

func (i *IKIOSocket) onMessage(pkt ikio.Packet, wc ikio.WriteCloser) {
	switch pkt := pkt.(type) {
	case *RPCNegoPacket:
	case *RPCPacket:
		switch pkt.Type() {
		case PacketTypeHint:
			i.logger.Debug(
				"event", "heartbeat", "remote", i.peerAddr,
				"local", i.localAddr,
			)
		case PacketTypeRequest:
			// TODO
		case PacketTypeResponse:
			value, ok := i.controller.Load(pkt.ID)
			if !ok {
				return
			}
			header := make(map[string]string)
			pkt.ForeachHeader(func(key, value []byte) error {
				header[string(key)] = string(value)
				return nil
			})
			c := &Context{
				Header: header,
				Body:   pkt.Payload,
			}
			i.contextEnd(value.(*requestContext), c, nil)
		default:
			// TODO
		}
	}
}

func (i *IKIOSocket) onConnect(wc ikio.WriteCloser) bool {
	_, err := wc.Write(context.TODO(), NegoPacket)
	if err != nil {
		i.logger.Debug(
			"event", "onconnect", "remote", i.peerAddr,
			"local", i.localAddr, "err", err,
		)
		return false
	}
	i.logger.Debug(
		"event", "onconnect", "remote", i.peerAddr,
		"local", i.localAddr,
	)
	if channel, ok := wc.(*ikio.ClientChannel); ok {
		channel.RunAfter(10*time.Second, 10*time.Second, func(t time.Time, wc ikio.WriteCloser) {
			wc.Write(context.TODO(), HintPacket)
		})
	}
	return true
}

func (i *IKIOSocket) onClose(wc ikio.WriteCloser) {
	i.Close()
}

func buildPacket(request *requestContext) *RPCPacket {
	pkt := new(RPCPacket)
	pkt.Tp = PacketTypeRequest
	pkt.ID = request.seq
	for key, value := range request.reqContext.Header {
		pkt.AddHeader([]byte(key), []byte(value))
	}
	pkt.Payload = request.reqContext.Body
	return pkt
}

func (i *IKIOSocket) contextEnd(request *requestContext, c *Context, err error) {
	if !atomic.CompareAndSwapInt64(&request.end, 0, 1) {
		// context already end
		return
	}

	i.controller.Delete(request.seq)
	request.err = err
	request.context = c
	request.timer.Stop()
	request.wg.Done()
}

func (i *IKIOSocket) dispatch() {
	defer func() { i.wg.Done() }()
	for {
		select {
		case <-i.exitC:
			// exit
			return
		case request := <-i.requestC:
			i.controller.Store(request.seq, request)
			if _, err := i.wc.Write(context.TODO(), buildPacket(request)); err != nil {
				i.contextEnd(request, nil, err)
			}
		}
	}
}
