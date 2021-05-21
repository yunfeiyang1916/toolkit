package ini

import (
	ini "github.com/gookit/ini/parser"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder"
)

type iniEncoder struct{}

func (i iniEncoder) Encode(v interface{}) ([]byte, error) {
	b, err := ini.Encode(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (i iniEncoder) Decode(d []byte, v interface{}) error {
	return ini.Decode(d, v)
}

func (i iniEncoder) String() string {
	return "ini"
}

// NewEncoder is ini encoder
func NewEncoder() encoder.Encoder {
	return iniEncoder{}
}
