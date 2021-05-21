package pb

import (
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/yunfeiyang1916/toolkit/ecode"
)

func (e *Error) Error() string {
	return strconv.FormatInt(int64(e.GetDmError()), 10)
}

// Code is the code of error.
func (e *Error) Code() int {
	return int(e.GetDmError())
}

// Message is error message.
func (e *Error) Message() string {
	return e.GetErrMsg()
}

// Equal compare whether two errors are equal.
func (e *Error) Equal(ec error) bool {
	return ecode.Cause(ec).Code() == e.Code()
}

// From will convert ecode.Codes to pb.Error.
func From(ec ecode.Codes) *Error {
	return &Error{
		DmError: proto.Int(ec.Code()),
		ErrMsg:  proto.String(ec.Message()),
	}
}
