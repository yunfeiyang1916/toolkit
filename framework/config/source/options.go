package source

import (
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/toml"
	"golang.org/x/net/context"
)

type Options struct {
	Encoder encoder.Encoder
	Context context.Context
}

type Option func(o *Options)

func NewOptions(opts ...Option) Options {
	options := Options{
		Encoder: toml.NewEncoder(),
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}
	return options
}

func WithEncoder(e encoder.Encoder) Option {
	return func(o *Options) {
		o.Encoder = e
	}
}
