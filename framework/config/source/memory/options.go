package memory

import (
	"github.com/yunfeiyang1916/toolkit/framework/config/source"
	"golang.org/x/net/context"
)

type rawChangeSetKey struct{}
type jsonChangeSetKey struct{}
type tomlChangeSetKey struct{}

// WithChangeSet allows a changeSet to be set
func WithChangeSet(cs *source.ChangeSet) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, rawChangeSetKey{}, cs)
	}
}

// WithData allows the source data to be set
func WithDataJson(d []byte) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, jsonChangeSetKey{}, &source.ChangeSet{
			Data:   d,
			Format: "json",
		})
	}
}

func WithDataToml(d []byte) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, tomlChangeSetKey{}, &source.ChangeSet{
			Data:   d,
			Format: "toml",
		})
	}
}
