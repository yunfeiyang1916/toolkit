package upstream

import (
	"hash/fnv"
	"unsafe"

	"golang.org/x/net/context"
)

type hashCtxKeyType struct{}

var (
	hashCtxKey = hashCtxKeyType{}
)

func NewContextWithHash(ctx context.Context, hash interface{}) context.Context {
	return context.WithValue(ctx, hashCtxKey, hashCode(hash))
}

func NewContextWithHashCode(ctx context.Context, hash uint32) context.Context {
	return context.WithValue(ctx, hashCtxKey, hash)
}

func HashFromContext(ctx context.Context) (code uint32, ok bool) {
	code, ok = ctx.Value(hashCtxKey).(uint32)
	return
}

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
