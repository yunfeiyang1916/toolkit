package ecode

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/pkg/errors"
)

var (
	messages sync.Map             // map[int]string
	codes    = map[int]struct{}{} // register codes.
	mux      sync.Mutex
)
var (
	OK        = Int(0)
	ParamErr  = Int(499)
	ServerErr = Int(500)
)

// Register register ecode message map.
func Register(cm map[int]string) {
	for k, v := range cm {
		messages.Store(k, v)
	}
}

// New new a ecode.Codes by int value.
// NOTE: ecode must unique in global, the New will check repeat and then panic.
func New(e int) Code {
	return add(e)
}

// Error returns a  ecode.Codes and register associated ecode message
// NOTE: Error codes and messages should be kept together.
// ecode must unique in global, the Error will check repeat and then panic.
func Error(e int, msg string) Code {
	code := add(e)
	Register(map[int]string{
		e: msg,
	})
	return code
}
func add(e int) Code {
	if _, ok := codes[e]; ok {
		fmt.Printf("ecode: %d already exist \n", e)
	}
	codes[e] = struct{}{}
	return Int(e)
}

// Codes ecode error interface which has a code & message.
type Codes interface {
	// Error return Code in string form
	Error() string
	// Code get error code.
	Code() int
	// Message get code message.
	Message() string
	// Equal for compatible.
	Equal(error) bool
}

// A Code is an int error code spec.
type Code int

func (e Code) Error() string {
	return strconv.FormatInt(int64(e), 10)
}

// Code return error code
func (e Code) Code() int { return int(e) }

// Message return error message
func (e Code) Message() string {
	v, ok := messages.Load(e.Code())
	if !ok {
		return e.Error()
	}
	ret, _ := v.(string)
	return ret
}

// Equal for compatible.
func (e Code) Equal(err error) bool { return EqualError(e, err) }

// Int parse code int to error.
func Int(i int) Code { return Code(i) }

// String parse code string to error.
func String(e string) Code {
	if e == "" {
		return Int(0)
	}
	// try error string
	i, err := strconv.Atoi(e)
	if err != nil {
		return Int(500)
	}
	return Code(i)
}

// Cause cause from error to ecode.
func Cause(e error) Codes {
	if e == nil {
		return Int(0)
	}
	ec, ok := errors.Cause(e).(Codes)
	if ok {
		return ec
	}
	return String(e.Error())
}

// Equal equal a and b by code int.
func Equal(a, b Codes) bool {
	if a == nil {
		a = Int(0)
	}
	if b == nil {
		b = Int(0)
	}
	return a.Code() == b.Code()
}

// EqualError equal error
func EqualError(code Codes, err error) bool {
	return Cause(err).Code() == code.Code()
}
