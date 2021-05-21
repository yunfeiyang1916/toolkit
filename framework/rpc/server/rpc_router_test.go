package server

import (
	"errors"

	"golang.org/x/net/context"

	//"github.com/yunfeiyang1916/toolkit/framework/rpc/codec"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Exported struct {
}

func (*Exported) Method1(_ context.Context, req *int) (*int, error) {
	return nil, nil
}

func (*Exported) unexportedM(_ context.Context, req *int) (*int, error) {
	return nil, nil
}

func (*Exported) Method2(_ context.Context, req *int) (*int, error) {
	return nil, nil
}

func (*Exported) Method3(_ context.Context, req *int) {
}

func (*Exported) Method4(_ context.Context, req *int) (int, error) {
	return 0, nil
}

func (*Exported) Method5(_ context.Context, reply *int) int {
	return 0
}

func (*Exported) Method6(_ context.Context, req *int) (int, error) {
	return 0, nil
}

func TestPrepareMethod(t *testing.T) {
	var tests = map[string]struct {
		err error
	}{
		"Method1": {
			err: nil,
		},
		"Method2": {},
		"Method3": {
			err: errors.New("method Method3 of func(*server.Exported, context.Context, *int) has wrong number of outs: 0"),
		},
		"Method4": {
			err: errors.New("method Method4 reply type not a pointer: int"),
		},
		"Method5": {
			err: errors.New("method Method5 of func(*server.Exported, context.Context, *int) int has wrong number of outs: 1"),
		},
		"Method6": {
			err: errors.New("method Method6 reply type not a pointer: int"),
		},
	}

	typ := reflect.TypeOf(&Exported{})
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			method, ok := typ.MethodByName(name)
			assert.True(ok)
			_, err := prepareMethod(method)
			assert.Equal(test.err, err)
		})
	}
}

type TestHandler struct {
}

type Request struct {
}

type Response struct {
	Age int
}

func (*TestHandler) Call(_ context.Context, req Request, reply *Response) error {
	return nil
}

func TestRouterServe(t *testing.T) {

}

func BenchmarkCall(b *testing.B) {
	r := newRouter()
	r.Handle(r.NewHandler(new(TestHandler)))
	b.ResetTimer()
	s, m, args, err := r.signature("TestHandler", "Call")
	if err != nil {
		b.Fatal(err)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := r.call(context.TODO(), s, m, args)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
