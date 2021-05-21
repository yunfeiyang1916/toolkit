package retry

import (
	"fmt"
	"strings"

	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"golang.org/x/net/context"
)

type RetryError struct {
	RawErrors []error
	Final     error
}

func (e RetryError) Error() string {
	var suffix string
	if len(e.RawErrors) > 1 {
		a := make([]string, len(e.RawErrors)-1)
		for i := 0; i < len(e.RawErrors)-1; i++ { // last one is Final
			a[i] = e.RawErrors[i].Error()
		}
		suffix = fmt.Sprintf(" (previously: %s)", strings.Join(a, "; "))
	}
	return fmt.Sprintf("%v%s", e.Final, suffix)
}

type BreakError struct {
	Err error
}

func (b BreakError) Error() string {
	return b.Err.Error()
}

type KeepTrying interface {
	KeepTrying(n int, received error) bool
	maxTimes(n int)
}

type trymax struct {
	max int
}

func (t *trymax) KeepTrying(n int, received error) bool {
	return n < t.max
}

func (t *trymax) maxTimes(n int) {
	t.max = n
}

func Retry(max int) core.Plugin {
	return retryKeepTrying(&trymax{max: max})
}

type internalKey struct{}

var Key = internalKey{}

func retryKeepTrying(keep KeepTrying) core.Plugin {
	return core.Function(func(ctx context.Context, c core.Core) {
		r := ctx.Value(Key)
		if r != nil && r.(int) > 0 {
			keep.maxTimes(r.(int))
		}
		var final RetryError
		idx := c.Index()
		for i := 0; ; i++ {
			c.Reset(idx)
			c.Next(ctx)
			err := c.Err()
			if err == nil {
				c.Abort()
				return
			}
			switch e := err.(type) {
			case BreakError:
				c.AbortErr(e.Err)
				return
			}

			final.RawErrors = append(final.RawErrors, err)
			final.Final = err

			if keep.KeepTrying(i, err) {
				continue
			}
			break
		}
		c.AbortErr(final)
	})
}
