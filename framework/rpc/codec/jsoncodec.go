package codec

import (
	"encoding/json"
)

type jsonCodec struct{}

func NewJSONCodec() Codec {
	return jsonCodec{}
}

func (jsonCodec) Encode(request interface{}) ([]byte, error) {
	return json.Marshal(request)
}

func (jsonCodec) Decode(body []byte, response interface{}) error {
	return json.Unmarshal(body, response)
}
