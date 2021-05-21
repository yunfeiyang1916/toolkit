package ikiosocket

import (
	"net"
	"time"

	"github.com/yunfeiyang1916/toolkit/ikio"
	"github.com/yunfeiyang1916/toolkit/logging"
	"golang.org/x/net/context"
)

type servercb func(string, *Context) (*Context, error)

type Server struct {
	servercb servercb
	*ikio.Server
	*logging.Logger
}

func NewServer(l *logging.Logger, cb servercb) *Server {
	server := &Server{}
	lg := logging.New()
	lg.SetOutputByName("logs/ikio.log")
	lg.SetLevelByString("error")
	ikioserver := ikio.NewServer(
		ikio.WithLogger(lg),
		ikio.OnConnectOption(server.connect),
		ikio.OnCloseOption(server.disconnect),
		ikio.WorkerSizeOption(128),
		ikio.TimerBufferSizeOption(128),
		ikio.WriterBufferSizeOption(128),
		ikio.HandlerBufferSizeOption(128),
		ikio.CustomCodecOption(func() ikio.Codec { return &RPCCodec{} }),
	)

	ikioserver.Register(PacketTypeRequest, server.request, ikio.HandlePooledRandom)
	ikioserver.Register(PacketTypeResponse, server.response, ikio.HandlePoolNewRoutine)
	ikioserver.Register(PacketTypeHint, server.hint, ikio.HandlePooledRandom)
	server.Server = ikioserver
	server.servercb = cb
	server.Logger = l
	return server
}

func (s *Server) Start(ln net.Listener) error {
	return s.Server.Start(ln)
}

func (s *Server) Stop() {
	s.Server.Stop()
}

func (s *Server) connect(wc ikio.WriteCloser) bool {
	_, err := wc.Write(context.TODO(), NegoPacket)
	if err != nil {
		return false
	}
	conn := wc.(*ikio.ServerChannel)
	conn.RunAfter(
		30*time.Second,
		30*time.Second,
		func(t time.Time, wc ikio.WriteCloser) {
			if time.Now().UnixNano()-conn.LastActive() > (time.Second * 30).Nanoseconds() {
				s.Debug(
					"event", "runafter",
					"connid", conn.ConnID(),
					"remote", conn.RemoteAddr(),
					"lastactive", time.Unix(0, conn.LastActive()),
				)
				wc.Close()
			}
		},
	)
	return true
}

func (s *Server) disconnect(wc ikio.WriteCloser) {
	conn := wc.(*ikio.ServerChannel)
	if time.Now().UnixNano()-conn.LastActive() < (time.Second * 1).Nanoseconds() {
		return
	}
	s.Debug(
		"event", "disconnect",
		"connid", conn.ConnID(),
		"remote", conn.RemoteAddr(),
		"lastactive", time.Unix(0, conn.LastActive()),
	)
}

func (s *Server) request(ctx context.Context, wc ikio.WriteCloser) {
	conn := wc.(*ikio.ServerChannel)
	message := ikio.MessageFromContext(ctx).(*RPCPacket)
	header := map[string]string{}
	message.ForeachHeader(func(key, value []byte) error {
		header[string(key)] = string(value)
		return nil
	})
	req := &Context{
		Header: header,
		Body:   message.Payload,
	}
	response, err := s.servercb(conn.RemoteAddr().String(), req)
	if err != nil {
		s.Debug("connid", conn.ConnID(), "remote", conn.RemoteAddr(), "event", "callback", "err", err)
		return
	}
	resppkt := message
	resppkt.Header = nil
	for key, value := range response.Header {
		resppkt.AddHeader([]byte(key), []byte(value))
	}
	resppkt.Payload = response.Body
	resppkt.Tp = PacketTypeResponse
	_, err = wc.Write(ctx, resppkt)
	if err != nil {
		s.Debug("connid", conn.ConnID(), "remote", conn.RemoteAddr(), "event", "write", "err", err)
	}
	Put(message)
}

func (s *Server) response(ctx context.Context, wc ikio.WriteCloser) {
	s.Debug("event", "response")
}

func (s *Server) hint(ctx context.Context, wc ikio.WriteCloser) {
	// conn := wc.(*ikio.ServerChannel)
	message := ikio.MessageFromContext(ctx)
	wc.Write(ctx, message)
	// s.Log("event", "hint", "connid", conn.ConnID(), "remote", conn.RemoteAddr())
}
