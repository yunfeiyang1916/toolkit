package main

import (
	"context"
	"log"
	"net"
	pb "server/user/v1"

	"google.golang.org/grpc"
)

type UserService struct {
	pb.UnimplementedUserServer
}

func (u *UserService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterReply, error) {
	log.Println("接收到来着客户端的消息：", req)
	return &pb.RegisterReply{
		Name:   req.Name,
		Gender: req.Gender,
		Mobile: req.Mobile,
	}, nil
}

func (u *UserService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginReply, error) {
	log.Println("接收到来着客户端的消息：", req)
	return &pb.LoginReply{
		Mobile: req.Mobile,
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatal(err)
	}
	// 创建grpc服务
	s := grpc.NewServer()
	// 注册服务
	pb.RegisterUserServer(s, &UserService{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
