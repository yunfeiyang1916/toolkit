package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPluginGeneralCase(t *testing.T) {
	signature := ""
	c := New(nil)
	c.Use(Function(func(ctx context.Context, c Core) {
		signature += "A"
		c.Next(ctx)
		signature += "B"
	}), Function(func(_ context.Context, c Core) {
		signature += "C"
	})).Use(Function(func(_ context.Context, core Core) {
		signature += "D"
	}))
	c.Next(context.TODO())
	assert.Equal(t, "ACDB", signature)
}

func TestPluginAbort(t *testing.T) {
	signature := ""
	c := New(nil)
	c.Use(Function(func(ctx context.Context, c Core) {
		signature += "A"
		c.Next(ctx)
		signature += "B"
		assert.True(t, c.IsAborted())
	}), Function(func(ctx context.Context, c Core) {
		signature += "C"
		c.Abort()
		c.Next(ctx)
		signature += "D"
	})).Use(Function(func(_ context.Context, c Core) {
		signature += "E"
	}))
	c.Next(context.TODO())
	assert.Equal(t, "ACDB", signature)
}

func TestPluginAbortErr(t *testing.T) {
	signature := ""
	c := New(nil)
	c.Use(Function(func(ctx context.Context, c Core) {
		signature += "A"
		c.Next(ctx)
		signature += "B"
		assert.True(t, c.IsAborted())
		assert.NotNil(t, c.Err())
	}), Function(func(ctx context.Context, c Core) {
		signature += "C"
		c.AbortErr(errors.New(""))
		c.Next(ctx)
		signature += "D"
	})).Use(Function(func(_ context.Context, c Core) {
		signature += "E"
	}))
	c.Next(context.TODO())
	assert.NotNil(t, c.Err())
	assert.Equal(t, "ACDB", signature)
}

func TestPluginCopy(t *testing.T) {
	signature := ""
	c := New(nil)
	c.Use(Function(func(ctx context.Context, c Core) {
		signature += "A"
		c.Copy().Next(ctx)
		c.Next(ctx)
		signature += "B"
	}), Function(func(ctx context.Context, c Core) {
		signature += "C"
		c.Next(ctx)
		signature += "D"
	})).Use(Function(func(_ context.Context, c Core) {
		signature += "E"
	}))
	c.Next(context.TODO())
	assert.Equal(t, "ACEDCEDB", signature)
}

func TestPluginCopyWithAbortErr(t *testing.T) {
	signature := ""
	flage := false
	c := New(nil)
	c.Use(Function(func(ctx context.Context, c Core) {
		signature += "A"
		c2 := c.Copy()
		c2.Next(ctx)
		assert.NotNil(t, c2.Err())
		c.Next(ctx)
		signature += "B"
	}), Function(func(ctx context.Context, c Core) {
		if !flage {
			c.AbortErr(errors.New(""))
			flage = true
			return
		}
		signature += "C"
		c.Next(ctx)
		signature += "D"
	})).Use(Function(func(_ context.Context, c Core) {
		signature += "E"
	}))
	c.Next(context.TODO())
	assert.Nil(t, c.Err())
	assert.Equal(t, "ACEDB", signature)
}
