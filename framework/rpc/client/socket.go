package client

import (
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/internal/ikiosocket"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/internal/metadata"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/internal/rpcerror"
	"github.com/yunfeiyang1916/toolkit/logging"
)

// socket interface, for mock.
type socket interface {
	Call(endpoint string, header map[string]string, body []byte) ([]byte, error)
	Close() error
}

// ikRPCSocket is a wraper of IKIOSocket,
// and implement socket interface.
type ikRPCSocket struct {
	socket *ikiosocket.IKIOSocket
	req    time.Duration
}

func newIKSocket(logger *logging.Logger, conn net.Conn, req time.Duration) *ikRPCSocket {
	socket := ikiosocket.New(logger)
	socket.StartWithConn(conn)
	return &ikRPCSocket{
		req:    req,
		socket: socket,
	}
}

func (r *ikRPCSocket) Call(endpoint string, header map[string]string, body []byte) ([]byte, error) {
	reqmeta := &metadata.RpcMeta{
		Type:       metadata.RpcMeta_REQUEST.Enum(),
		SequenceId: proto.Uint64(uint64(0)),
		Method:     proto.String(endpoint),
		Failed:     proto.Bool(false),
		ErrorCode:  proto.Int32(int32(rpcerror.Success)),
		//Reason:    proto.String(desc),
	}
	metabody, err := proto.Marshal(reqmeta)
	if err != nil {
		return nil, rpcerror.Errorf(rpcerror.Internal, "binary socket: %s", err)
	}

	h := map[string]string{}
	for k, v := range header {
		h[k] = v
	}
	h[metadata.MetaHeaderKey] = string(metabody)
	h[metadata.DataHeaderKey] = string(body)

	response, err := r.socket.Call(&ikiosocket.Context{
		Header: h,
		Body:   nil,
	}, ikiosocket.Timeout(r.req))

	if err != nil {
		if err == ikiosocket.ErrExited {
			return nil, err
		}
		if err == ikiosocket.ErrTimeout {
			return nil, rpcerror.Errorf(rpcerror.Timeout, "binary socket: %s", err)
		}
		return nil, rpcerror.Errorf(rpcerror.Internal, "binary socket: %s", err)
	}

	if _, ok := response.Header[metadata.MetaHeaderKey]; !ok {
		return nil, rpcerror.New(rpcerror.Internal, "binary socket: missing meta header")
	}

	respmeta := &metadata.RpcMeta{}
	if err := proto.Unmarshal([]byte(response.Header[metadata.MetaHeaderKey]), respmeta); err != nil {
		return nil, rpcerror.Errorf(rpcerror.Internal, "binary socket: %s", err)
	}

	if respmeta.GetType() != metadata.RpcMeta_RESPONSE {
		return nil, rpcerror.Errorf(rpcerror.Internal, "binary socket: unexpected message type %s", respmeta.GetType())
	}

	if respmeta.GetFailed() {
		return nil, rpcerror.New(int(respmeta.GetErrorCode()), respmeta.GetReason())
	}
	return []byte(response.Header[metadata.DataHeaderKey]), nil
}

func (r *ikRPCSocket) Close() error {
	return r.socket.Close()
}
