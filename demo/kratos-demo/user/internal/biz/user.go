package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

type User struct {
	ID       int64
	Username string
	Mobile   string
	Nickname string
	Avatar   string
	Password string
}

// UserRepo 仓储层接口，使用依赖倒置的原则，接口定义在domain层，实现在data层
type UserRepo interface {
	Save(context.Context, *User) (*User, error)
	Update(context.Context, *User) (*User, error)
	FindByID(context.Context, int64) (*User, error)
	ListByName(context.Context, string) ([]*User, error)
	FindByMobile(context.Context, string) (*User, error)
	ListAll(context.Context) ([]*User, error)
}

type UserUsecase struct {
	repo UserRepo
	log  *log.Helper
}

func NewUserUsecase(repo UserRepo, logger log.Logger) *UserUsecase {
	return &UserUsecase{repo: repo, log: log.NewHelper(logger)}
}

func CreateUser() {

}
