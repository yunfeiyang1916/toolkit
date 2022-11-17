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
	err := r.data.gormDB.WithContext(ctx).Save(entity).Error
	return entity, err
}

func (r *userRepo) Update(ctx context.Context, entity *biz.User) (*biz.User, error) {
	return entity, nil
}

func (r *userRepo) FindByID(ctx context.Context, id int64) (*biz.User, error) {
	var entity biz.User
	err := r.data.gormDB.WithContext(ctx).Where("id=?", id).First(&entity).Error
	return &entity, err
}

func (r *userRepo) FindByMobile(ctx context.Context, mobile string) (*biz.User, error) {
	var entity biz.User
	err := r.data.gormDB.WithContext(ctx).Where("mobile=?", mobile).First(&entity).Error
	return &entity, err
}

func (r *userRepo) ListByName(context.Context, string) ([]*biz.User, error) {
	return nil, nil
}

func (r *userRepo) ListAll(context.Context) ([]*biz.User, error) {
	return nil, nil
}
