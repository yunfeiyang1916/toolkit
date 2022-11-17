package service

import (
	"context"
	"time"
	"user/internal/biz"

	pb "user/api/user/v1"
)

type UserService struct {
	pb.UnimplementedUserServer
	uc *biz.UserUsecase
}

func NewUserService(uc *biz.UserUsecase) *UserService {
	return &UserService{uc: uc}
}

func (s *UserService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterReply, error) {
	entity := &biz.User{
		Name:       req.Name,
		Gender:     req.Gender,
		Mobile:     req.Mobile,
		Password:   req.Password,
		CreateTime: time.Now(),
	}
	entity.UpdateTime = entity.CreateTime
	entity, err := s.uc.Register(ctx, entity)
	if err != nil {
		return nil, err
	}
	res := &pb.RegisterReply{
		Name:     entity.Name,
		Gender:   entity.Gender,
		Mobile:   entity.Mobile,
		Password: entity.Password,
	}
	return res, nil
}
func (s *UserService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginReply, error) {
	return &pb.LoginReply{}, nil
}
func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserReply, error) {
	return &pb.CreateUserReply{}, nil
}
func (s *UserService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserReply, error) {
	return &pb.UpdateUserReply{}, nil
}
func (s *UserService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserReply, error) {
	return &pb.DeleteUserReply{}, nil
}
func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserReply, error) {
	return &pb.GetUserReply{}, nil
}
func (s *UserService) ListUser(ctx context.Context, req *pb.ListUserRequest) (*pb.ListUserReply, error) {
	return &pb.ListUserReply{}, nil
}
