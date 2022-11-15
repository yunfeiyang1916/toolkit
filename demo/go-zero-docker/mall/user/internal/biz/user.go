package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type User struct {
	Id         int64
	Name       string
	Gender     int64
	Mobile     string
	Password   string
	CreateTime time.Time
	UpdateTime time.Time
}

// UserRepo 仓储层接口，使用依赖倒置的原则，接口定义在domain层，实现在data层
type UserRepo interface {
	Save(context.Context, *User) (*User, error)
	Update(context.Context, *User) (*User, error)
	FindByID(context.Context, int64) (*User, error)
	ListByName(context.Context, string) ([]*User, error)
	ListAll(context.Context) ([]*User, error)
}

type UserUsecase struct {
	repo UserRepo
	log  *log.Helper
}

func NewUserUsecase(repo UserRepo, logger log.Logger) *UserUsecase {
	return &UserUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *UserUsecase) CreateUser(ctx context.Context, entity *User) (*User, error) {
	uc.log.WithContext(ctx).Infof("CreateUser:%v", entity.Id)
	return uc.repo.Save(ctx, entity)
}
