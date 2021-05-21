package client

import (
	"time"

	"golang.org/x/net/context"
)

type Option func(*Options)

// CallOption used by Invoke
type CallOption func(*CallOptions)

var (
	DefaultPoolSize       = 5
	DefaultPoolTTL        = time.Minute
	DefaultDialTimeout    = time.Second * 5
	DefaultRetries        = 1
	DefaultRequestTimeout = time.Second * 1
)

type Client interface {
	Invoke(context.Context, interface{}, interface{}, ...CallOption) error
}
