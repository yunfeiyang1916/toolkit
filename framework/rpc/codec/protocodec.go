package codec

import (
	"github.com/golang/protobuf/proto"
)

type protoCodec struct{}

func NewProtoCodec() Codec {
	return protoCodec{}
}

func (protoCodec) Encode(request interface{}) ([]byte, error) {
	return proto.Marshal(request.(proto.Message))
}

func (protoCodec) Decode(body []byte, response interface{}) error {
	return proto.Unmarshal(body, response.(proto.Message))
}
