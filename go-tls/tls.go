package tls

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"

	context "golang.org/x/net/context"
)

type contextKey struct{}

var (
	contextKeyActive = contextKey{}
)

// GoID get an uniq id, not as the same goroutine id
func GoID() int64 {
	return int64(uintptr(G()))
}

// GoID get goroutine id, this method is very slow
func GoIDSlow() int64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseInt(string(b), 10, 64)
	return n

}

const (
	maxShardsCount = 128
)

var (
	tlsStore []tlsStorage
)

type tlsStorage = sync.Map

func init() {
	tlsStore = make([]tlsStorage, maxShardsCount)
	for i := 0; i < maxShardsCount; i++ {
		tlsStore[i] = tlsStorage{}
	}
}

func getLocal(goid int64) map[interface{}]interface{} {
	shardIndex := goid % maxShardsCount
	gls, found := tlsStore[shardIndex].Load(goid)
	if found {
		return gls.(map[interface{}]interface{})
	}
	return nil
}

func setLocal(goid int64, local map[interface{}]interface{}) {
	shardIndex := goid % maxShardsCount
	tlsStore[shardIndex].Store(goid, local)
}

func deleteLocal(goid int64) {
	shardIndex := goid % maxShardsCount
	tlsStore[shardIndex].Delete(goid)

}

//Set k v store, k v not thread safe
func Set(k, v interface{}) {
	goid := GoID()
	shardIndex := goid % maxShardsCount
	glsI, _ := tlsStore[shardIndex].LoadOrStore(goid, make(map[interface{}]interface{}))
	glsI.(map[interface{}]interface{})[k] = v
}

//Delete k v store, k v not thread safe
func Delete(k interface{}) {
	local := getLocal(GoID())
	delete(local, k)
}

func Get(k interface{}) (v interface{}, exist bool) {
	local := getLocal(GoID())
	v, exist = local[k]
	return
}

func SetContext(ctx context.Context) {
	Set(contextKeyActive, ctx)
}

func GetContext() (context.Context, bool) {
	v, exist := Get(contextKeyActive)
	if !exist {
		return nil, false
	}
	ctx, ok := v.(context.Context)
	return ctx, ok
}

func DeleteContext() {
	Delete(contextKeyActive)
}

//Flush clear this goroutine tls store
func Flush() {
	deleteLocal(GoID())
}

func For(ctx context.Context, f func()) func() {
	return func() {
		local := make(map[interface{}]interface{})
		local[contextKeyActive] = ctx
		goid := GoID()
		setLocal(goid, local)
		defer deleteLocal(goid)
		f()
	}
}

//With will copy parent tls, not thread safe
func With(f func()) func() {
	parent := getLocal(GoID())
	child := make(map[interface{}]interface{})
	for k, v := range parent {
		child[k] = v
	}
	return func() {
		goid := GoID()
		setLocal(goid, child)
		defer deleteLocal(goid)
		f()
	}
}

//Wrap auto copy parent context
func Wrap(f func()) func() {
	ctx, exist := GetContext()
	return func() {
		local := make(map[interface{}]interface{})
		if exist {
			local[contextKeyActive] = ctx
		}
		goid := GoID()
		setLocal(goid, local)
		defer deleteLocal(goid)
		f()
	}
}
