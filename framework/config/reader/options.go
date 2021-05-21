package reader

import (
	"strings"

	"github.com/yunfeiyang1916/toolkit/framework/config/encoder"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/ini"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/json"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/toml"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/xml"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/yaml"
)

// Encoding represents supported encoder
var Encoding map[string]encoder.Encoder

// init define supported format(encoder)
func init() {
	Encoding = map[string]encoder.Encoder{
		"json": json.NewEncoder(),
		"yaml": yaml.NewEncoder(),
		"toml": toml.NewEncoder(),
		"xml":  xml.NewEncoder(),
		"yml":  yaml.NewEncoder(),
		"ini":  ini.NewEncoder(),
	}
}

// WithEncoder use for add a new format
func WithEncoder(e encoder.Encoder) {
	Encoding[e.String()] = e
}

func Encoder(format string) encoder.Encoder {
	enc := toml.NewEncoder()
	if e := Encoding[strings.ToLower(format)]; e != nil {
		enc = e
	}
	return enc
}
