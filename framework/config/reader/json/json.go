package json

import (
	"errors"
	"time"

	"github.com/yunfeiyang1916/toolkit/framework/config/encoder"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/json"
	"github.com/yunfeiyang1916/toolkit/framework/config/reader"
	"github.com/yunfeiyang1916/toolkit/framework/config/source"
	"github.com/yunfeiyang1916/toolkit/logging"

	"github.com/imdario/mergo"
)

type jsonReader struct {
	json encoder.Encoder
}

// NewReader creates a json reader
func NewReader() reader.Reader {
	return &jsonReader{
		json: json.NewEncoder(),
	}
}

// implement Reader interface
func (j *jsonReader) Merge(changes ...*source.ChangeSet) (*source.ChangeSet, error) {
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
			codec = j.json
		}
		var data map[string]interface{}
		if err := codec.Decode(m.Data, &data); err != nil {
			logging.GenLogf("jsonReader on Merge, decode to map failed, err: %v, data: %s", err, string(m.Data))
			return nil, err
		}
		if err := mergo.Map(&merged, data, mergo.WithOverride); err != nil {
			logging.GenLogf("jsonReader on Merge, merge map failed, err: %v, data: %+v", err, data)
			return nil, err
		}
	}

	b, err := j.json.Encode(merged)
	if err != nil {
		logging.GenLogf("jsonReader on Merge, encode merged data failed, err: %v, data: %+v", err, merged)
		return nil, err
	}

	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Data:      b,
		Source:    "json",
		Format:    j.json.String(),
	}
	cs.Checksum = cs.Sum()
	return cs, nil
}

func (j *jsonReader) Values(ch *source.ChangeSet) (reader.Values, error) {
	if ch == nil {
		return nil, errors.New("changeSet is nil")
	}
	if ch.Format != "json" {
		return nil, errors.New("unsupported format")
	}
	v, err := newValues(ch)
	if err != nil {
		logging.GenLogf("jsonReader on Values, read failed, err: %v, data: %s", err, string(ch.Data))
	}
	return v, err
}

func (j *jsonReader) String() string {
	return "json"
}
