package toml

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/yunfeiyang1916/toolkit/framework/config/reader"
	"github.com/yunfeiyang1916/toolkit/framework/config/source"
)

type tomlValues struct {
	ch   *source.ChangeSet
	meta *toml.MetaData
	data interface{} // after translate
}

func newValues(ch *source.ChangeSet) (reader.Values, error) {
	var tmp interface{}
	meta, err := toml.DecodeReader(bytes.NewBuffer(ch.Data), &tmp)
	if err != nil {
		return nil, err
	}
	data := translate(tmp)
	return &tomlValues{data: data, meta: &meta, ch: ch}, nil
}

func (tm *tomlValues) Bytes() []byte {
	b := bytes.NewBuffer(tm.ch.Data)
	return b.Bytes()
}

func (tm *tomlValues) String() string {
	b := bytes.NewBuffer(tm.ch.Data)
	return b.String()
}

func (tm *tomlValues) Get(keys ...string) reader.Value {
	if !tm.meta.IsDefined(keys...) {
		return &tomlValue{data: tm.data}
	}

	if len(keys) == 0 {
		return &tomlValue{data: tm.data}
	}

	m := tm.data.(map[string]interface{})
	var t interface{}
	for _, key := range keys {
		t = m[key]
		mm, ok := m[key].(map[string]interface{})
		if ok {
			m = mm
			continue
		}
		break
	}

	return &tomlValue{data: t}
}

func (tm *tomlValues) Map() map[string]interface{} {
	tmp := tm.data.(map[string]interface{})
	return tmp
}

func (tm *tomlValues) Range(f func(k string, v interface{})) {
	m := tm.Map()
	for k, v := range m {
		kk, vv := k, v
		f(kk, vv)
	}
}

func (tm *tomlValues) Scan(v interface{}) error {
	_, err := toml.DecodeReader(bytes.NewBuffer(tm.ch.Data), v)
	if err != nil {
		return err
	}
	return nil
}

type tomlValue struct {
	data interface{}
}

func (tv *tomlValue) Bool(def bool) bool {
	if v, ok := tv.data.(bool); ok {
		return v
	}
	str, ok := tv.data.(string)
	if !ok {
		return def
	}

	b, err := strconv.ParseBool(str)
	if err != nil {
		return def
	}
	return b
}

func (tv *tomlValue) Int(def int) int {
	switch t := tv.data.(type) {
	case int:
		return t
	case uint:
		return int(t)
	case int32:
		return int(t)
	case uint32:
		return int(t)
	case int64:
		return int(t)
	case uint64:
		return int(t)
	case float32:
		return int(t)
	case float64:
		return int(t)
	}

	str, ok := tv.data.(string)
	if !ok {
		return def
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return def
	}
	return i
}

func (tv *tomlValue) String(def string) string {
	if v, ok := tv.data.(string); ok {
		return v
	}
	return def
}

func (tv *tomlValue) Float64(def float64) float64 {
	if v, ok := tv.data.(float64); ok {
		return v
	}
	str, ok := tv.data.(string)
	if !ok {
		return def
	}
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return def
	}
	return f
}

func (tv *tomlValue) Duration(def time.Duration) time.Duration {
	vv, ok := tv.data.(string)
	if !ok {
		return def
	}
	value, err := time.ParseDuration(vv)
	if err != nil {
		return def
	}
	return value
}

func (tv *tomlValue) StringSlice(def []string) []string {
	m, ok := tv.data.([]interface{})
	if !ok {
		return def
	}
	res := make([]string, len(m))
	for i, v := range m {
		res[i] = fmt.Sprintf("%v", v)
	}
	if len(res) == 0 {
		return def
	}
	return res
}

func (tv *tomlValue) StringMap(def map[string]string) map[string]string {
	m, ok := tv.data.(map[string]interface{})
	if !ok {
		return def
	}

	res := map[string]string{}
	for k, v := range m {
		res[k] = fmt.Sprintf("%v", v)
	}
	if len(res) == 0 {
		return def
	}
	return res
}

func (tv *tomlValue) Scan(val interface{}) error {
	b := bytes.NewBuffer(tv.Bytes())
	_, err := toml.DecodeReader(b, val)
	return err
}

func (tv *tomlValue) Bytes() []byte {
	b := bytes.NewBuffer([]byte{})
	err := toml.NewEncoder(b).Encode(tv.data)
	if err != nil {
		return nil
	}
	return b.Bytes()
}

func translate(tomlData interface{}) interface{} {
	switch orig := tomlData.(type) {
	case map[string]interface{}:
		typed := make(map[string]interface{}, len(orig))
		for k, v := range orig {
			typed[k] = translate(v)
		}
		return typed
	case []map[string]interface{}:
		typed := make([]map[string]interface{}, len(orig))
		for i, v := range orig {
			typed[i] = translate(v).(map[string]interface{})
		}
		return typed
	case []interface{}:
		typed := make([]interface{}, len(orig))
		for i, v := range orig {
			typed[i] = translate(v)
		}
		return typed
	case time.Time:
		return orig.Format("2006-01-02T15:04:05Z")
	case bool:
		return orig
	case int64:
		return orig
	case float64:
		return orig
	case string:
		return orig
	}

	panic(fmt.Sprintf("Unknown type: %T", tomlData))
}
