package consul

import (
	"github.com/yunfeiyang1916/toolkit/framework/config/source"
	"golang.org/x/net/context"
)

type addressKey struct{}
type prefixKey struct{}
type stripPrefixKey struct{}

// WithAddress sets the consul address
func WithAddress(a string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, addressKey{}, a)
	}
}

// WithAbsPath sets the key prefix to use
func WithAbsPath(p string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, prefixKey{}, p)
	}
}

// UsePrefix indicates whether to remove the prefix from config entries, or leave it in place.
func UsePrefix(strip bool) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, stripPrefixKey{}, strip)
	}
}
