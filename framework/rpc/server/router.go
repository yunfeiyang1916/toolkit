package server

import (
	"errors"
	"fmt"

	"golang.org/x/net/context"

	"reflect"
	"sync"
	"unicode"
	"unicode/utf8"
)

var (

	// Precompute the reflect type for error. Can't use error directly
	// because Typeof takes an empty interface value. This is annoying.
	typeOfError = reflect.TypeOf((*error)(nil)).Elem()
)

type service struct {
	name   string                 // name of service
	rcvr   reflect.Value          // receiver of methods for the service
	typ    reflect.Type           // type of the receiver
	method map[string]*methodType // registered methods
}

type methodType struct {
	sync.Mutex  // protects counters
	method      reflect.Method
	ArgType     reflect.Type
	ReplyType   reflect.Type
	ContextType reflect.Type
}

func (m *methodType) prepareContext(ctx context.Context) reflect.Value {
	if contextv := reflect.ValueOf(ctx); contextv.IsValid() {
		return contextv
	}
	return reflect.Zero(m.ContextType)
}

type router struct {
	serviceMap map[string]*service
}

func newRouter() *router {
	return &router{
		serviceMap: make(map[string]*service),
	}
}

func (r *router) NewHandler(handler interface{}, opts ...HandlerOption) Handler {
	return newRpcHandler(handler, opts...)
}

func (r *router) Handle(h Handler) error {
	if len(h.Name()) == 0 {
		return errors.New("rpc.Handle: handler has no name")
	}
	hdlr := reflect.ValueOf(h.Handler())
	name := reflect.Indirect(hdlr).Type().Name()

	if !isExported(name) {
		return errors.New("rpc.Handle: type " + name + " is not exported")
	}
	// check name
	if _, present := r.serviceMap[h.Name()]; present {
		return errors.New("rpc.Handle: service already defined: " + h.Name())
	}

	rcvr := h.Handler()

	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = h.Name()
	s.method = make(map[string]*methodType)

	// Install the methods
	for m := 0; m < s.typ.NumMethod(); m++ {
		method := s.typ.Method(m)
		if mt, _ := prepareMethod(method); mt != nil {
			s.method[method.Name] = mt
		}
	}

	// Check there are methods
	if len(s.method) == 0 {
		return errors.New("rpc Register: type " + s.name + " has no exported methods of suitable type")
	}

	// save handler
	r.serviceMap[s.name] = s
	return nil
}

func (r *router) GetPaths() []string {
	var paths []string
	for k, v := range r.serviceMap {
		for kk := range v.method {
			paths = append(paths, fmt.Sprintf("%s.%s", k, kk))
		}
	}
	return paths
}

func (r *router) call(ctx context.Context, s *service, m *methodType, args interface{}) (interface{}, error) {
	function := m.method.Func
	returnValues := function.Call(
		[]reflect.Value{
			s.rcvr,                // receiver
			m.prepareContext(ctx), // context
			reflect.ValueOf(args), // request
			//		reflect.ValueOf(reply), // response
		},
	)
	// The return value for the method is an error.
	if err := returnValues[1].Interface(); err != nil {
		return nil, err.(error)
	}
	return returnValues[0].Interface(), nil
}

func (r *router) signature(service, method string) (s *service, m *methodType, args interface{}, err error) {
	s = r.serviceMap[service]
	if s == nil {
		err = errors.New("rpc: can't find service " + service)
		return
	}
	m = s.method[method]
	if m == nil {
		err = errors.New("rpc: can't find method " + method)
		return
	}

	// Decode the argument value.
	var argv reflect.Value
	if m.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType)
	}

	args = argv.Interface()
	return
}

/*
func (r *router) Serve(ctx context.Context, codec codec.Codec, request *rpcRequest) (*rpcResponse, error) {

	// argv guaranteed to be a pointer now.
	if err := codec.Decode(request.body, argv.Interface()); err != nil {
		return nil, err
	}

	if err := r.call(ctx, service, mtype, argv, replyv); err != nil {
		return nil, err
	}

	body, err := codec.Encode(replyv.Interface())
	if err != nil {
		return nil, err
	}

	return &rpcResponse{body: body}, nil
}
*/

func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// prepareMethod returns a methodType for the provided method or nil
// in case if the method was unsuitable.
func prepareMethod(method reflect.Method) (*methodType, error) {
	mtype := method.Type
	mname := method.Name
	var replyType, argType, contextType reflect.Type

	// Method must be exported.
	if method.PkgPath != "" {
		return nil, errors.New("method must be exported")
	}

	switch mtype.NumIn() {
	case 3:
		// method that takes a context
		argType = mtype.In(2)
		contextType = mtype.In(1)
	default:
		return nil, fmt.Errorf("method %s of %s has wrong number of parameters: %d", mname, mtype, mtype.NumIn())
	}

	// Method needs one out.
	switch mtype.NumOut() {
	case 2:
		replyType = mtype.Out(0)
	default:
		return nil, fmt.Errorf("method %s of %s has wrong number of outs: %d", mname, mtype, mtype.NumOut())
	}

	// First arg need not be a pointer.
	if !isExportedOrBuiltinType(argType) {
		return nil, fmt.Errorf("%s argument type not exported: %s", mname, argType)
	}

	if argType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("method %s arg type not a pointer: %s", mname, argType)
	}

	if replyType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("method %s reply type not a pointer: %s", mname, replyType)
	}

	// Reply type must be exported.
	if !isExportedOrBuiltinType(replyType) {
		return nil, fmt.Errorf("method %s reply type not exported: %s", mname, replyType)
	}

	// The return type of the method must be error.
	if returnType := mtype.Out(1); returnType != typeOfError {
		return nil, fmt.Errorf("method %s returns %s not error", mname, returnType)
	}
	return &methodType{method: method, ArgType: argType, ReplyType: replyType, ContextType: contextType}, nil
}
