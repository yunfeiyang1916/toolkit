package biz

import (
	"errors"

	"github.com/google/wire"
	"gorm.io/gorm"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewGreeterUsecase, NewUserUsecase)

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
