package ecode

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/yunfeiyang1916/toolkit/framework/breaker"
	"github.com/yunfeiyang1916/toolkit/framework/ratelimit"
	"github.com/yunfeiyang1916/toolkit/go-upstream/circuit"
)

const (
	Success            = 0
	ChannelBroken      = 9
	FailedPrecondition = 100
	Unavailable        = 101
	Canceled           = 102
	DeadlineExceeded   = 103
	ConfigLb           = 104
	BreakerOpen        = 105
	LimitedExceeded    = 106
	UnknownNamespace   = 107
	UnKnown            = 1001
)

var (
	ErrClientLB         = errors.New("host not found from upstream")
	ErrUnknownNamespace = errors.New("unknown namespace")
)

func ConvertErr(err error) int {
	code := UnKnown
	switch err.(type) {
	case nil:
		code = Success
	case *url.Error:
		code = Unavailable
	case circuit.BreakerError:
		code = BreakerOpen
	default:
		if strings.Contains(err.Error(), "no living upstream") {
			return ConfigLb
		}
		switch err {
		case io.EOF:
			code = ChannelBroken
		case io.ErrClosedPipe, io.ErrNoProgress, io.ErrShortBuffer, io.ErrShortWrite, io.ErrUnexpectedEOF:
			code = FailedPrecondition
		case context.DeadlineExceeded:
			code = DeadlineExceeded
		case context.Canceled:
			code = Canceled
		case ErrClientLB:
			code = ConfigLb
		case circuit.ErrMaxConcurrent, circuit.ErrRateLimit, circuit.ErrSystemLoad, circuit.ErrAverageRT, circuit.ErrConsecutive, circuit.ErrPercent, circuit.ErrOpen:
			code = BreakerOpen
		case breaker.ErrOpen, breaker.ErrConsecutiveThreshold, breaker.ErrPercentThreshold:
			code = BreakerOpen
		case ratelimit.ErrLimited:
			code = LimitedExceeded
		case ErrUnknownNamespace:
			code = UnknownNamespace
		}
	}
	return code
}

func ConvertHttpStatus(err error) int {
	code := ConvertErr(err)
	switch code {
	case Success:
		return http.StatusOK
	case LimitedExceeded, BreakerOpen, UnknownNamespace:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}
