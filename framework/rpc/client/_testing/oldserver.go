package _test

//import (
//	old "github.com/yunfeiyang1916/toolkit/framework/rpc/client/_testing/old"
//	rpc "git.inke.cn/inkelogic/rpc-go"
//	"golang.org/x/net/context"
//)
//
//type Server struct {
//	s *rpc.Server
//}
//
//func (s *Server) Echo(ctx context.Context, r *old.EchoRequest) (*old.EchoResponse, error) {
//	return &old.EchoResponse{
//		Response: r.Message,
//		Code:     old.ResponseCode_SUCCESS.Enum(),
//	}, nil
//}
//
//func New() *Server {
//	s := &Server{}
//	type config struct{}
//	rpc.NewConfigToml("_testing/config.toml", &config{})
//	server := rpc.NewServer()
//	old.RegisterEchoServiceServer(server, s)
//	s.s = server
//	return s
//}
//
//func (s *Server) Start(port int) error {
//	return s.s.Serve(port)
//}
