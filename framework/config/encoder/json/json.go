package json

import (
	"bytes"

	jsoniter "github.com/json-iterator/go"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type jsonEncoder struct{}

func (j jsonEncoder) Encode(v interface{}) ([]byte, error) {
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	if err := jsonEncoder.Encode(v); err != nil {
		return nil, err
	}
	return bf.Bytes(), nil
}

func (j jsonEncoder) Decode(d []byte, v interface{}) error {
	return json.Unmarshal(d, v)
}

func (j jsonEncoder) String() string {
	return "json"
}

func NewEncoder() encoder.Encoder {
	return jsonEncoder{}
}
