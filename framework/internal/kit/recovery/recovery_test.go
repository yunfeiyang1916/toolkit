package recovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"golang.org/x/net/context"
)

func TestRecovery(t *testing.T) {
	assert := assert.New(t)
	c := core.New(nil)
	c.Use(Recovery(true))
	c.Use(core.Function(func(ctx context.Context, c core.Core) {
		panic("test")
	}))
	c.Next(context.TODO())
	assert.NotNil(c.Err())
}
