package sd

import (
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
)

type Factory interface {
	Factory(host string) (core.Plugin, error)
}
