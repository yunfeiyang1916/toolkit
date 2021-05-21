package ikio

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"time"
	"unsafe"

	"github.com/yunfeiyang1916/toolkit/logging"
	"golang.org/x/net/context"
)

// ErrUndefined for undefined message type.
type ErrUndefined int32

func (e ErrUndefined) Error() string {
	return fmt.Sprintf("undefined message type %d", e)
}

// Error codes returned by failures dealing with server or connection.
var (
	ErrWouldBlock   = errors.New("would block")
	ErrServerClosed = errors.New("server has been closed")
)

// some constants.
const (
	defaultWorkersNum     = 10
	defaultWriteBuffSize  = 32
	defaultHandleBuffSize = 16
	defaultTimerBuffSize  = 8
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type onConnectFunc func(WriteCloser) bool
type onPacketFunc func(Packet, WriteCloser)
type onCloseFunc func(WriteCloser)
type onErrorFunc func(WriteCloser)

type workerFunc func()
type onScheduleFunc func(time.Time, WriteCloser)

// Hashable is a interface for hashable object.
type Hashable interface {
	HashCode() int32
}

const intSize = unsafe.Sizeof(1)

func hashCode(k interface{}) uint32 {
	var code uint32
	h := fnv.New32a()
	switch v := k.(type) {
	case bool:
		h.Write((*((*[1]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case int:
		h.Write((*((*[intSize]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case int8:
		h.Write((*((*[1]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case int16:
		h.Write((*((*[2]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case int32:
		h.Write((*((*[4]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case int64:
		h.Write((*((*[8]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case uint:
		h.Write((*((*[intSize]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case uint8:
		h.Write((*((*[1]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case uint16:
		h.Write((*((*[2]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case uint32:
		h.Write((*((*[4]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case uint64:
		h.Write((*((*[8]byte)(unsafe.Pointer(&v))))[:])
		code = h.Sum32()
	case string:
		h.Write([]byte(v))
		code = h.Sum32()
	case Hashable:
		c := v.HashCode()
		h.Write((*((*[4]byte)(unsafe.Pointer(&c))))[:])
		code = h.Sum32()
	default:
		panic("key not hashable")
	}
	return code
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	kd := rv.Type().Kind()
	switch kd {
	case reflect.Ptr, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func printStack() {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	os.Stderr.Write(buf[:n])
}

// ContextKey is the key type for putting context-related data.
type serverContextKey struct{}
type messageContextKey struct{}
type connIDContextKey struct{}

// Context keys for messge, server and net ID.
var (
	messageCtxKey = messageContextKey{}
	serverCtxKey  = serverContextKey{}
	connIDCtxKey  = connIDContextKey{}
)

// NewContextWithServer returns a new Context that carries server.
func NewContextWithServer(ctx context.Context, s *Server) context.Context {
	return context.WithValue(ctx, serverCtxKey, s)
}

// NewContextWithMessage returns a new Context that carries message.
func NewContextWithMessage(ctx context.Context, msg Packet) context.Context {
	return context.WithValue(ctx, messageCtxKey, msg)
}

func MessageFromContext(ctx context.Context) Packet {
	return ctx.Value(messageCtxKey).(Packet)
}

func NewContextWithConnID(ctx context.Context, connID int64) context.Context {
	return context.WithValue(ctx, connIDCtxKey, connID)
}

func ConnIDFromContext(ctx context.Context) int64 {
	return ctx.Value(connIDCtxKey).(int64)
}

func ServerFromContext(ctx context.Context) *Server {
	return ctx.Value(serverCtxKey).(*Server)
}

func normalnizeOptions(opt ...Option) options {
	var opts options
	for _, o := range opt {
		o(&opts)
	}
	if opts.workerSize <= 0 {
		opts.workerSize = defaultWorkersNum
	}
	if opts.writerBufferSize <= 0 {
		opts.writerBufferSize = defaultWriteBuffSize
	}
	if opts.handlerBufferSize <= 0 {
		opts.handlerBufferSize = defaultHandleBuffSize
	}
	if opts.timerBufferSize <= 0 {
		opts.timerBufferSize = defaultTimerBuffSize
	}
	if opts.logger == nil {
		opts.logger = logging.New() // stderr
	}
	if opts.writeTimeout == 0 {
		opts.writeTimeout = time.Duration(30 * time.Second)
	}
	return opts
}
