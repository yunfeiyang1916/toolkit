package main

import (
	pb "client/user/v1"
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

func main() {
	// 与服务器建立连接
	conn, err := grpc.Dial("127.0.0.1:9000", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	userClient := pb.NewUserClient(conn)
	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 向服务器发送消息
	r, err := userClient.Register(ctx, &pb.RegisterRequest{Name: "张三"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("收到响应：%s", r.Name)
}
