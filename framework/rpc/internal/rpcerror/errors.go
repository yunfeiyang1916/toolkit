package rpcerror

import (
	"fmt"
)

type RPCError interface {
	Error() string
	Code() int
}

type defaultError struct {
	message string
	code    int
}

func (r *defaultError) Error() string {
	return r.message
}

func (r *defaultError) Code() int {
	return r.code
}

func New(code int, message string) RPCError {
	return &defaultError{message, code}
}

func Errorf(code int, format string, a ...interface{}) RPCError {
	if code == Success {
		return nil
	}
	return &defaultError{
		code:    code,
		message: fmt.Sprintf(format, a...),
	}
}

func Error(code int, err error) RPCError {
	return Errorf(code, "%s", err)
}
