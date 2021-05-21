package ikio

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yunfeiyang1916/toolkit/ikio/timer"
	"golang.org/x/net/context"

	log "github.com/yunfeiyang1916/toolkit/logging"
	metrics "github.com/yunfeiyang1916/toolkit/metrics"
)

var valueKey = struct{}{}

type options struct {
	codec             codecNewFunc
	onConnect         onConnectFunc
	onMessage         onPacketFunc
	onMessagePoolType HandlePoolType
	onClose           onCloseFunc
	workerSize        int // numbers of worker go-routines
	timerBufferSize   int // size of buffered channel
	handlerBufferSize int // size of buffered channel
	writerBufferSize  int // size of buffered channel
	readTimeout       time.Duration
	writeTimeout      time.Duration
	logger            *log.Logger

	metricTags []interface{}
}

// OnTimeOut represents a timed task.
type OnTimeOut struct {
	Callback      func(time.Time, WriteCloser)
	Ctx           context.Context
	AskForWorkers bool
}

// NewOnTimeOut returns OnTimeOut.
func NewOnTimeOut(ctx context.Context, cb func(time.Time, WriteCloser)) *OnTimeOut {
	return &OnTimeOut{
		Callback: cb,
		Ctx:      ctx,
	}
}

// HandlePoolType handle pool type
type HandlePoolType int8

// handler type
const (
	HandleNoPooled HandlePoolType = iota
	HandlePooledRandom
	HandlePooledStick
	HandlePoolNewRoutine
)

// Option sets server options.
type Option func(*options)

// MetricsTags SetOption metrics tags
func MetricsTags(kvs ...interface{}) Option {
	return func(o *options) {
		o.metricTags = kvs
	}
}

// WithLogger SetOption Logger
func WithLogger(l *log.Logger) Option {
	return func(o *options) {
		o.logger = l
	}
}

// ReadTimeoutOption SetSocket read timeout
func ReadTimeoutOption(t time.Duration) Option {
	return func(o *options) {
		o.readTimeout = t
	}
}

// WriteTimeoutOption Option SetSocket write timeout
func WriteTimeoutOption(t time.Duration) Option {
	return func(o *options) {
		o.writeTimeout = t
	}
}

// CustomCodecOption returns a Option that will apply a custom Codec.
func CustomCodecOption(codec func() Codec) Option {
	return func(o *options) {
		o.codec = codec
	}
}

// WorkerSizeOption returns a Option that will set the number of go-routines
// in WorkerPool.
func WorkerSizeOption(workerSz int) Option {
	return func(o *options) {
		o.workerSize = workerSz
	}
}

// WriterBufferSizeOption returns a Option that is the size of writer buffered channel,
func WriterBufferSizeOption(indicator int) Option {
	return func(o *options) {
		o.writerBufferSize = indicator
	}
}

// HandlerBufferSizeOption returns a Option that is the size of handler buffered channel,
func HandlerBufferSizeOption(indicator int) Option {
	return func(o *options) {
		o.handlerBufferSize = indicator
	}
}

// TimerBufferSizeOption returns a Option that is the size of timer buffered channel,
func TimerBufferSizeOption(indicator int) Option {
	return func(o *options) {
		o.timerBufferSize = indicator
	}
}

// OnConnectOption returns a Option that will set callback to call when new
// client connected.
func OnConnectOption(cb func(WriteCloser) bool) Option {
	return func(o *options) {
		o.onConnect = cb
	}
}

// OnMessageOption returns a Option that will set callback to call when new
// message arrived.
func OnMessageOption(cb func(Packet, WriteCloser), poolType ...HandlePoolType) Option {
	return func(o *options) {
		o.onMessage = cb
		if len(poolType) > 0 {
			o.onMessagePoolType = poolType[0]
		} else {
			o.onMessagePoolType = HandleNoPooled
		}
	}
}

// OnCloseOption returns a Option that will set callback to call when client
// closed.
func OnCloseOption(cb func(WriteCloser)) Option {
	return func(o *options) {
		o.onClose = cb
	}
}

type Server struct {
	opts            options
	ctx             context.Context
	cancel          context.CancelFunc
	wg              *sync.WaitGroup
	conns           *sync.Map
	codec           Codec
	mu              sync.Mutex
	lis             map[net.Listener]bool
	messageRegistry map[int32]*handlerEntry
	timing          *timer.TimingWheel
	identifier      int64
	logger          *log.Logger
}

type handlerEntry struct {
	handlerFunc     HandlerFunc
	handlerPoolType HandlePoolType
}

func NewServer(opt ...Option) *Server {
	opts := normalnizeOptions(opt...)
	// initiates go-routine pool instance
	initGlobalWorkPool(opts.workerSize)
	s := &Server{
		opts:            opts,
		conns:           &sync.Map{},
		wg:              &sync.WaitGroup{},
		lis:             make(map[net.Listener]bool),
		messageRegistry: make(map[int32]*handlerEntry),
		identifier:      0,
		logger:          opts.logger,
	}
	s.ctx, s.cancel = context.WithCancel(NewContextWithServer(context.Background(), s))
	s.timing = timer.NewTimingWheel(s.ctx, timer.MetricsTags(opts.metricTags...))
	return s
}

func (s *Server) Start(l net.Listener) error {
	s.mu.Lock()
	if s.lis == nil {
		s.mu.Unlock()
		l.Close()
		return ErrServerClosed
	}
	s.lis[l] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if s.lis != nil && s.lis[l] {
			l.Close()
			delete(s.lis, l)
		}
		s.mu.Unlock()
	}()
	s.logger.Infof("server start, net %s addr %s", l.Addr().Network(), l.Addr().String())
	s.wg.Add(1)
	go s.timeOutLoop()

	/*
		go func() {
			for {
				time.Sleep(time.Second * 1)
				select {
				case <-s.ctx.Done():
					return
				default:
					metrics.Gauge("ikio.server.connections", len(s.lis))
				}
			}
		}()
	*/

	var tempDelay time.Duration
	for {
		rawConn, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay >= max {
					tempDelay = max
				}
				s.logger.Warnf("accept error %s, retrying in %d", err, tempDelay)
				select {
				case <-time.After(tempDelay):
				case <-s.ctx.Done():
				}
				continue
			}
			return err
		}
		tempDelay = 0
		connID := atomic.AddInt64(&s.identifier, 1)
		conn := NewServerChannel(connID, s, rawConn)
		s.conns.Store(connID, conn)
		s.wg.Add(1)
		go conn.Start()
	}
}
func (s *Server) Stop() {
	// immediately stop accepting new clients
	s.mu.Lock()
	listeners := s.lis
	s.lis = nil
	s.mu.Unlock()

	for l := range listeners {
		l.Close()
		s.logger.Infof("stop accepting at address %s", l.Addr().String())
	}

	// close all connections
	conns := map[int64]*ServerChannel{}

	s.conns.Range(func(k, v interface{}) bool {
		i := k.(int64)
		c := v.(*ServerChannel)
		conns[i] = c
		return true
	})
	for _, c := range conns {
		c.conn.Close()
	}

	s.mu.Lock()
	s.cancel()
	s.mu.Unlock()

	s.wg.Wait()

}

// Handler takes the responsibility to handle incoming messages.
type Handler interface {
	Handle(context.Context, WriteCloser)
}

// HandlerFunc serves as an adapter to allow the use of ordinary functions as handlers.
type HandlerFunc func(context.Context, WriteCloser)

// Handle calls f(ctx, c)
func (f HandlerFunc) Handle(ctx context.Context, c WriteCloser) {
	f(ctx, c)
}

func (s *Server) Register(msgType int32, handler HandlerFunc, handleType HandlePoolType) {
	if _, ok := s.messageRegistry[msgType]; ok {
		panic(fmt.Sprintf("trying to register message %d twice", msgType))
	}

	s.messageRegistry[msgType] = &handlerEntry{
		handlerFunc:     handler,
		handlerPoolType: handleType,
	}
}

func (s *Server) getHandlerFunc(msgType int32) *handlerEntry {
	entry, ok := s.messageRegistry[msgType]
	if !ok {
		return nil
	}
	return entry
}

func (s *Server) timeOutLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			return
		case ctx := <-s.timing.TimeOutChannel():
			on := ctx.Value().(*OnTimeOut)
			connID := ConnIDFromContext(on.Ctx)
			beginTime := time.Now()
			if v, ok := s.conns.Load(connID); ok {
				sc := v.(*ServerChannel)
				sc.addTimeout(ctx)
			} else {
			}
			metrics.Timer("ikio.server.timer-block", beginTime, s.opts.metricTags...)
		}
	}
}

// Broadcast broadcasts message to all server connections managed.
func (s *Server) Broadcast(ctx context.Context, msg Packet, except func(connid int64) bool) {
	s.conns.Range(func(k, v interface{}) bool {
		c := v.(*ServerChannel)
		if except(c.ConnID()) {
			return true
		}
		if _, err := c.Write(ctx, msg); err != nil {
			return true
		}
		return true
	})
}

// Unicast unicasts message to a specified conn.
func (s *Server) Unicast(ctx context.Context, id int64, msg Packet) (int, error) {
	v, ok := s.conns.Load(id)
	if ok {
		return v.(*ServerChannel).Write(ctx, msg)
	}
	return 0, fmt.Errorf("conn id %d not found", id)
}

// Conn returns a server connection with specified ID.
func (s *Server) Conn(id int64) (*ServerChannel, bool) {
	v, ok := s.conns.Load(id)
	if ok {
		return v.(*ServerChannel), ok
	}
	return nil, ok
}
