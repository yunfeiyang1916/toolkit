package codec

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	metadata "github.com/yunfeiyang1916/toolkit/framework/rpc/internal/metadata"
)

func TestProtoEncode(t *testing.T) {
	codec := NewProtoCodec()
	assert := assert.New(t)

	d := new(metadata.RpcMeta)
	d.Type = metadata.RpcMeta_REQUEST.Enum()
	d.SequenceId = proto.Uint64(1)
	_, err := codec.Encode(d)
	assert.Nil(err)
}

func TestProtoDecode(t *testing.T) {
	codec := NewProtoCodec()
	assert := assert.New(t)

	d := new(metadata.RpcMeta)
	d.Type = metadata.RpcMeta_REQUEST.Enum()
	d.SequenceId = proto.Uint64(1)
	body, err := codec.Encode(d)
	assert.Nil(err)

	p := new(metadata.RpcMeta)
	err = codec.Decode(body, p)
	assert.Nil(err)
}
