package toml

import (
	"errors"
	"time"

	"github.com/yunfeiyang1916/toolkit/logging"

	"github.com/imdario/mergo"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/toml"
	"github.com/yunfeiyang1916/toolkit/framework/config/reader"
	"github.com/yunfeiyang1916/toolkit/framework/config/source"
)

type tomlReader struct {
	tm encoder.Encoder
}

// NewReader is a reader with toml encoder
func NewReader() reader.Reader {
	return &tomlReader{
		tm: toml.NewEncoder(),
	}
}

// Merge represents merge data by toml
func (t *tomlReader) Merge(changes ...*source.ChangeSet) (*source.ChangeSet, error) {
	var merged map[string]interface{}
	for _, m := range changes {
		if m == nil {
			continue
		}
		if len(m.Data) == 0 {
			continue
		}
		codec, ok := reader.Encoding[m.Format]
		if !ok {
			codec = t.tm
		}
		var data map[string]interface{}
		if err := codec.Decode(m.Data, &data); err != nil {
			logging.GenLogf("tomlReader on Merge, decode to map failed, err: %v, data: %s", err, string(m.Data))
			return nil, err
		}
		// merge map data
		if err := mergo.Map(&merged, data, mergo.WithOverride); err != nil {
			logging.GenLogf("tomlReader on Merge, merge map failed, err: %v, data: %+v", err, data)
			return nil, err
		}
	}

	b, err := t.tm.Encode(merged)
	if err != nil {
		logging.GenLogf("tomlReader on Merge, encode merged data failed, err: %v, data: %+v", err, merged)
		return nil, err
	}
	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Data:      b,
		Source:    "toml",
		Format:    t.tm.String(),
	}
	cs.Checksum = cs.Sum()
	return cs, nil
}

// Values implements reader.Value
func (t *tomlReader) Values(ch *source.ChangeSet) (reader.Values, error) {
	if ch == nil {
		return nil, errors.New("changeSet is nil")
	}
	if ch.Format != "toml" {
		return nil, errors.New("unsupported format")
	}
	v, err := newValues(ch)
	if err != nil {
		logging.GenLogf("tomlReader on Values, read failed, err: %v, data: %s", err, string(ch.Data))
	}
	return v, err
}

// String represents "toml" format
func (t *tomlReader) String() string {
	return "toml"
}
