package data

import (
	"context"
	"user/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type userRepo struct {
	// 连接数据库客户端
	data *Data
	log  *log.Helper
}

func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{data: data, log: log.NewHelper(logger)}
}

func (r *userRepo) Save(ctx context.Context, entity *biz.User) (*biz.User, error) {
	return entity, nil
}

func (r *userRepo) Update(ctx context.Context, entity *biz.User) (*biz.User, error) {
	return entity, nil
}

func (r *userRepo) FindByID(context.Context, int64) (*biz.User, error) {
	return nil, nil
}

func (r *userRepo) ListByName(context.Context, string) ([]*biz.User, error) {
	return nil, nil
}

func (r *userRepo) ListAll(context.Context) ([]*biz.User, error) {
	return nil, nil
}
