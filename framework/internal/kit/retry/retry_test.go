package retry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"golang.org/x/net/context"
)

func TestRetryNil(t *testing.T) {
	times := 0
	c := core.New([]core.Plugin{Retry(3), core.Function(func(ctx context.Context, c core.Core) {
		times++
		switch times {
		case 1:
			c.AbortErr(errors.New("frist"))
		case 2:
			c.AbortErr(errors.New("second"))
		case 3:
		}
	})})
	c.Next(context.Background())
	assert.Nil(t, c.Err())
}

func TestRetryNotNil(t *testing.T) {
	times := 0

	errs := []error{
		errors.New("frist"),
		errors.New("second"),
		errors.New("third"),
	}

	c := core.New([]core.Plugin{Retry(2), core.Function(func(ctx context.Context, c core.Core) {
		times++
		switch times {
		case 1:
			c.AbortErr(errs[0])
		case 2:
			c.AbortErr(errs[1])
		case 3:
			c.AbortErr(errs[2])
		}
	})})
	c.Next(context.Background())
	assert.NotNil(t, c.Err())
	retry := c.Err().(RetryError)
	assert.Equal(t, len(retry.RawErrors), 3)
	for i, err := range retry.RawErrors {
		assert.Equal(t, err, errs[i])
	}
}

func TestRetryZero(t *testing.T) {
	times := 0

	errs := []error{
		errors.New("frist"),
	}

	c := core.New([]core.Plugin{Retry(0), core.Function(func(ctx context.Context, c core.Core) {
		times++
		switch times {
		case 1:
			c.AbortErr(errs[0])
		}
	})})
	c.Next(context.Background())
	assert.NotNil(t, c.Err())
	retry := c.Err().(RetryError)
	assert.Equal(t, len(retry.RawErrors), 1)
	for i, err := range retry.RawErrors {
		assert.Equal(t, err, errs[i])
	}
}
