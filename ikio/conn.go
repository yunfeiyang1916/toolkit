package ikio

import (
	"bufio"
	"net"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yunfeiyang1916/toolkit/ikio/timer"
	"github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/metrics"
	"golang.org/x/net/context"
)

type WriteCloser interface {
	Write(context.Context, Packet) (int, error)
	Close() error
}

type chanWrap struct {
	p []byte
	t timeWrap
}

// Channel represents a server connection to a TCP server, it implments Conn.
type Channel struct {
	connID          int64
	reader          *bufio.Reader
	writer          *bufio.Writer
	conn            net.Conn
	codec           Codec
	once            *sync.Once
	wg              *sync.WaitGroup
	sendCh          chan chanWrap
	handlerCh       chan MessageHandler
	nameMutex       sync.Mutex
	name            string
	parentCtx       context.Context
	ctx             context.Context
	cancel          context.CancelFunc
	lastActiveMutex sync.Mutex
	lastActive      int64
	opts            channelOption
	mu              sync.Mutex
	pending         []*timer.Context
	timing          *timer.TimingWheel
	timerCh         chan *timer.Context
	realChannel     WriteCloser
	logger          *logging.Logger
	closed          int32
}

type channelOption struct {
	options

	connID        int64
	waitGroup     *sync.WaitGroup
	timingWheel   *timer.TimingWheel
	getHandleFunc func(int32) (HandlerFunc, HandlePoolType)
}

// MessageHandler is a combination of message and its handler function.
type MessageHandler struct {
	message     Packet
	handler     HandlerFunc
	handlerType HandlePoolType
}

// NewChannel returns a new Channel
func NewChannel(ctx context.Context, conn net.Conn, opts channelOption) *Channel {
	c := &Channel{
		opts:        opts,
		connID:      opts.connID,
		conn:        conn,
		reader:      bufio.NewReader(conn),
		writer:      bufio.NewWriter(conn),
		codec:       opts.codec(),
		parentCtx:   ctx,
		once:        &sync.Once{},
		wg:          &sync.WaitGroup{},
		sendCh:      make(chan chanWrap, opts.writerBufferSize),
		handlerCh:   make(chan MessageHandler, opts.handlerBufferSize),
		timerCh:     make(chan *timer.Context, opts.timerBufferSize),
		lastActive:  time.Now().UnixNano(),
		pending:     []*timer.Context{},
		timing:      opts.timingWheel,
		realChannel: nil,
		closed:      0,
	}
	c.ctx, c.cancel = context.WithCancel(NewContextWithConnID(ctx, opts.connID))
	c.name = conn.RemoteAddr().String()
	c.realChannel = c
	c.logger = opts.logger
	return c
}

// ConnID returns net ID of server connection.
func (c *Channel) ConnID() int64 {
	return c.connID
}

// SetName sets name of server connection.
func (c *Channel) SetName(name string) {
	c.nameMutex.Lock()
	defer c.nameMutex.Unlock()
	c.name = name
}

// Name returns the name of server connection.
func (c *Channel) Name() string {
	c.nameMutex.Lock()
	defer c.nameMutex.Unlock()
	name := c.name
	return name
}

// SetLastActive sets the heart beats of server connection.
func (c *Channel) SetLastActive(heart int64) {
	c.lastActiveMutex.Lock()
	defer c.lastActiveMutex.Unlock()
	c.lastActive = heart
}

// LastActive returns the heart beats of server connection.
func (c *Channel) LastActive() int64 {
	c.lastActiveMutex.Lock()
	defer c.lastActiveMutex.Unlock()
	return c.lastActive
}

// Start starts the server connection, creating go-routines for reading,
// writing and handlng.
func (c *Channel) Start() {
	c.logger.Infof("conn #%d start, local %v, remote %v", c.connID, c.conn.LocalAddr(), c.conn.RemoteAddr())
	onConnect := c.opts.onConnect
	if onConnect != nil {
		onConnect(c.realChannel)
	}

	c.wg.Add(3)
	go c.readLoop(c.wg)
	go c.handleLoop(c.wg)
	go c.writeLoop(c.wg)
}

// Close gracefully closes the server connection. It blocked until all sub
// go-routines are completed and returned.
func (c *Channel) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return nil
	}
	c.once.Do(func() {
		c.logger.Infof("conn #%d end, local %v remote %v", c.connID, c.conn.LocalAddr(), c.conn.RemoteAddr())
		// close net.Conn, any blocked read or write operation will be unblocked and
		// return errors.
		if tc, ok := c.conn.(*net.TCPConn); ok {
			// avoid time-wait state
			tc.SetLinger(0)
		}
		c.conn.Close()

		// cancel readLoop, writeLoop and handleLoop go-routines.
		c.mu.Lock()
		c.cancel()
		pending := c.pending
		c.pending = nil
		c.mu.Unlock()

		// wait until all go-routines exited.
		c.wg.Wait()
		// callback on close
		onClose := c.opts.onClose
		if onClose != nil {
			c.logger.Infof("close callback, conn #%d, local %v, remote %v", c.connID, c.conn.LocalAddr(), c.conn.RemoteAddr())
			// remove connection from server
			onClose(c.realChannel)
		}
		for _, id := range pending {
			c.CancelTimer(id)
		}

		// close all channels and block until all go-routines exited.
		close(c.sendCh)
		close(c.handlerCh)
		close(c.timerCh)

		c.mu.Lock()
		c.realChannel = nil
		c.writer = nil
		c.mu.Unlock()

		// tell I'm done :( .
		if c.opts.waitGroup != nil {
			c.opts.waitGroup.Done()
		}
	})
	return nil
}

// Write writes a message to the client.
func (c *Channel) Write(ctx context.Context, packet Packet) (int, error) {
	return c.asyncWrite(ctx, packet)
}

// RemoteAddr returns the remote address of server connection.
func (c *Channel) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// LocalAddr returns the local address of server connection.
func (c *Channel) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// AddPendingTimer adds a new timer ID to client connection.
func (c *Channel) AddPendingTimer(ctx *timer.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pending != nil {
		c.pending = append(c.pending, ctx)
	}
}

// CancelTimer cancels a timer with the specified ID.
func (c *Channel) CancelTimer(ctx *timer.Context) {
	c.timing.CancelTimer(ctx)
}

func (c *Channel) RunAfter(delay time.Duration, interval time.Duration, cb func(time.Time, WriteCloser), askForWorks ...bool) *timer.Context {
	timeout := NewOnTimeOut(c.ctx, cb)
	if askForWorks != nil && askForWorks[0] {
		timeout.AskForWorkers = true
	}
	ctx, err := c.timing.AddTimer(delay, interval, timer.WithValue(timeout))
	if err == nil {
		c.AddPendingTimer(ctx)
	}
	return ctx
}

func (c *Channel) asyncWrite(ctx context.Context, m Packet) (n int, err error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return 0, ErrServerClosed
	}
	b, err := m.Serialize()
	if err != nil {
		return
	}
	if atomic.LoadInt32(&c.closed) == 1 {
		return 0, ErrServerClosed
	}
	defer func() {
		if p := recover(); p != nil {
			c.logger.Warnf("send fail, conn #%d, stack %q", c.connID, debug.Stack())
		}
	}()
	select {
	case c.sendCh <- chanWrap{b, getTimestamp(ctx)}:
		err = nil
	default:
		err = ErrWouldBlock
	}
	return
}

/* readLoop() blocking read from connection, deserialize bytes into message,
then find corresponding handler, put it into channel */
func (c *Channel) readLoop(wg *sync.WaitGroup) {
	defer func() {
		if p := recover(); p != nil {
			c.logger.Warnf("read fail, conn #%d, local %v, remote %v, stack %q", c.connID, c.conn.LocalAddr(), c.conn.RemoteAddr(), debug.Stack())
		}
		wg.Done()
		c.Close()
	}()
	for {
		select {
		case <-c.ctx.Done(): // connection closed
			return
		case <-c.parentCtx.Done(): // server closed
			return
		default:
			msg, err := c.codec.Decode(c.reader)
			now := time.Now()
			if err != nil {
				if _, ok := err.(ErrUndefined); ok {
					// update last active
					c.SetLastActive(now.UnixNano())
					continue
				}
				c.logger.Warnf("read abort, closing conn #%d, local %v, remote %v, decode packet error %v", c.connID, c.conn.LocalAddr(), c.conn.RemoteAddr(), err)
				return
			}
			c.SetLastActive(time.Now().UnixNano())
			var handler HandlerFunc
			handlerType := HandlePooledRandom
			if c.opts.getHandleFunc != nil {
				handler, handlerType = c.opts.getHandleFunc(msg.Type())
			}
			c.handlerCh <- MessageHandler{message: msg, handler: handler, handlerType: handlerType}
			metrics.Timer("ikio.server.handle-block", now, c.opts.metricTags...)
		}
	}
}

/* writeLoop() receive message from channel, serialize it into bytes,
then blocking write into connection */
func (c *Channel) writeLoop(wg *sync.WaitGroup) {
	defer func() {
		if p := recover(); p != nil {
			c.logger.Warnf("write fail, conn #%d, local %v, remote %v, stack %q", c.connID, c.conn.LocalAddr(), c.conn.RemoteAddr(), debug.Stack())
		}
	}()
	defer func() {
		// drain all pending messages before exit
	OuterFor:
		for {
			select {
			case w := <-c.sendCh:
				if c.writer == nil {
					break OuterFor
				}
				if w.p != nil {
					if _, err := c.writer.Write(w.p); err != nil {
						break OuterFor
					}
				}
			default:
				break OuterFor
			}
		}
		if c.writer != nil {
			c.writer.Flush()
		}
		wg.Done()
		c.Close()
	}()

	var shm [10][]byte
	for {
		select {
		case <-c.ctx.Done(): // connection closed
			return
		case <-c.parentCtx.Done(): // server closed
			return
		case w := <-c.sendCh:
			if w.p != nil {
				recvTime := time.Now()
				for j := 0; j < len(shm); j++ {
					shm[j] = shm[j][0:0] // 清空
				}
				shm[0] = w.p
				// 如果写缓存区还有数据，读出最多10个数据再次写入
				n := len(c.sendCh)
				if n > 0 {
					for i := 0; i < 9; i++ {
						if i < n {
							ww := <-c.sendCh
							shm[i+1] = ww.p
						}
					}
				}
				var err error
				shmBuf := net.Buffers(shm[:])
				sendTime := time.Now()
				c.conn.SetWriteDeadline(sendTime.Add(c.opts.writeTimeout))
				_, err = shmBuf.WriteTo(c.conn)
				metrics.Timer("ikio.network.write", sendTime, w.t.tags...)
				if err != nil {
					c.logger.Warnf("write abort, closing conn #%d, local %v, remote %v, err %v", c.connID, c.conn.LocalAddr(), c.conn.RemoteAddr(), err)
					return
				}
				if w.t.t != 0 {
					beginTime := time.Unix(0, w.t.t)
					metrics.Timer("ikio.server.write", beginTime, w.t.tags...)
					rCost := recvTime.Sub(beginTime)
					sCost := time.Now().Sub(sendTime)
					// slow log
					if rCost > 5*time.Millisecond || sCost > 200*time.Millisecond {
						c.logger.Infof("on writeLoop, write too slow, conn #%d, local %v, remote %v, sendChanLen %d, recvfromChan cost %d, send2socket cost %d",
							c.connID, c.conn.LocalAddr(), c.conn.RemoteAddr(), len(c.sendCh), rCost.Milliseconds(), sCost.Milliseconds())
					}
				}
			}
		}
	}
}

// handleLoop() - put handler or timeout callback into worker go-routines
func (c *Channel) handleLoop(wg *sync.WaitGroup) {
	var (
		hash uint32
	)
	ctx := c.ctx
	if c.connID == 0 {
		hash = hashCode(c.Name())
	} else {
		hash = hashCode(c.connID)
	}

	defer func() {
		if p := recover(); p != nil {
			c.logger.Warnf("handle fail, conn #%d, local %v, remote %v, stack %q", c.connID, c.conn.LocalAddr(), c.conn.RemoteAddr(), debug.Stack())
		}
		wg.Done()
		c.Close()
	}()

	for {
		select {
		case <-c.ctx.Done(): // connection closed
			return
		case <-c.parentCtx.Done(): // server closed
			return
		case msgHandler := <-c.handlerCh:
			var handleFunc func()
			code := hash
			msg, handler, handlerType := msgHandler.message, msgHandler.handler, msgHandler.handlerType
			if handler != nil {
				handleFunc = func() { handler(NewContextWithMessage(ctx, msg), c.realChannel); c.wg.Done() }
			} else {
				if c.opts.onMessage == nil {
					continue
				}
				handleFunc = func() { defer c.wg.Done(); c.opts.onMessage(msg, c.realChannel) }
				handlerType = c.opts.onMessagePoolType
			}
			if handlerType == HandlePooledRandom {
				code = 0
			}
			c.wg.Add(1)
			switch handlerType {
			case HandlePoolNewRoutine:
				go func() {
					defer func() {
						if err := recover(); err != nil {
							c.logger.Warnf("handle fail(HandlePoolNewRoutine), conn #%d, local %v, remote %v, stack %q", c.connID, c.conn.LocalAddr(), c.conn.RemoteAddr(), debug.Stack())
						}
					}()
					handleFunc()
				}()
			case HandlePooledRandom, HandlePooledStick:
				err := WorkerPoolInstance().Put(code, handleFunc)
				if err != nil {
					c.wg.Done()
				}
				continue
			default:
				handleFunc()
			}

		case ctx := <-c.timerCh:
			if ctx == nil {
				continue
			}
			on := ctx.Value().(*OnTimeOut)
			handleFunc := func() { on.Callback(time.Now(), c.realChannel); c.wg.Done() }
			c.wg.Add(1)
			if on.AskForWorkers {
				err := WorkerPoolInstance().Put(hash, handleFunc)
				if err != nil {
					c.wg.Done()
				}
				continue
			}
			handleFunc()
		}
	}
}

func (c *Channel) addTimeout(to *timer.Context) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return
	}
	c.timerCh <- to
}

// SetContextValue sets extra data to server connection.
func (c *Channel) SetContextValue(k, v interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ctx = context.WithValue(c.ctx, k, v)
}

// ContextValue gets extra data from server connection.
func (c *Channel) ContextValue(k interface{}) interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ctx.Value(k)
}

type ServerChannel struct {
	*Channel
}

func NewServerChannel(id int64, s *Server, c net.Conn) *ServerChannel {
	opts := channelOption{
		options:     s.opts,
		connID:      id,
		waitGroup:   s.wg,
		timingWheel: s.timing,
	}
	oldClose := opts.onClose
	opts.onClose = func(conn WriteCloser) {
		s.conns.Delete(id)
		if oldClose != nil {
			oldClose(conn)
		}
	}
	opts.getHandleFunc = func(msgType int32) (HandlerFunc, HandlePoolType) {
		entry := s.getHandlerFunc(msgType)
		if entry == nil {
			return nil, HandleNoPooled
		}
		return entry.handlerFunc, entry.handlerPoolType
	}
	channel := &ServerChannel{
		Channel: NewChannel(s.ctx, c, opts),
	}
	channel.Channel.realChannel = channel
	return channel
}

// Close asyncclose
func (c *ServerChannel) Close() error {
	c.logger.Infof("server closing, conn #%d, local %v, remote %v", c.connID, c.conn.LocalAddr(), c.conn.RemoteAddr())
	c.cancel()
	return nil
}

type ClientChannel struct {
	channelMutex sync.Mutex
	*Channel
}

func NewClientChannel(c net.Conn, opt ...Option) *ClientChannel {
	opts := normalnizeOptions(opt...)
	oldClose := opts.onClose
	clientChannel := new(ClientChannel)
	var newClientChannel func(c net.Conn) *Channel
	newClientChannel = func(c net.Conn) *Channel {
		ctx := context.Background()
		copts := channelOption{
			options:       opts,
			connID:        0,
			waitGroup:     nil,
			timingWheel:   timer.NewTimingWheel(ctx, timer.MetricsTags(opts.metricTags...)),
			getHandleFunc: nil,
		}
		copts.onClose = func(conn WriteCloser) {
			copts.timingWheel.Stop()
			if oldClose != nil {
				oldClose(conn)
			}
		}
		channel := NewChannel(ctx, c, copts)
		channel.timerCh = copts.timingWheel.TimeOutChannel()
		return channel
	}
	clientChannel.Channel = newClientChannel(c)
	clientChannel.realChannel = clientChannel
	return clientChannel
}

// Close asyncclose
func (c *ClientChannel) Close() error {
	c.logger.Infof("client closing, conn #%d, local %v, remote %v", c.connID, c.conn.LocalAddr(), c.conn.RemoteAddr())
	c.cancel()
	return nil
}
