package recovery

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/metrics"
)

// Recovery 异常恢复插件
func Recovery(recoverPanic bool) core.Plugin {
	return core.Function(func(ctx context.Context, c core.Core) {
		defer func() {
			if rc := recover(); rc != nil {
				logging.CrashLogf("recover panic info: %q, stacks info:\n%s", rc, debug.Stack())
				debug.PrintStack()
				metrics.Meter("plugin.recovery", 1)
				c.AbortErr(fmt.Errorf("%q", rc))
				// 是否继续抛出异常
				if !recoverPanic {
					panic(rc)
				}
			}
		}()
		c.Next(ctx)
	})
}
