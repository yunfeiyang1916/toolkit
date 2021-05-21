package toml

import (
	"bytes"

	"github.com/BurntSushi/toml"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder"
)

type tomlEncoder struct{}

func (t tomlEncoder) Encode(v interface{}) ([]byte, error) {
	b := bytes.NewBuffer([]byte{})
	err := toml.NewEncoder(b).Encode(v)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (t tomlEncoder) Decode(d []byte, v interface{}) error {
	return toml.Unmarshal(d, v)
}

func (t tomlEncoder) String() string {
	return "toml"
}

// NewEncoder is a toml encoder
func NewEncoder() encoder.Encoder {
	return tomlEncoder{}
}
