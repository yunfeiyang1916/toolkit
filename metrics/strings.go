package metrics

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"sync"
)

var (
	bufferPool sync.Pool
)

func builtinToString(k interface{}) string {
	switch m := k.(type) {
	case int8:
		return strconv.FormatInt(int64(m), 10)
	case uint8:
		return strconv.FormatInt(int64(m), 10)
	case int16:
		return strconv.FormatInt(int64(m), 10)
	case uint16:
		return strconv.FormatInt(int64(m), 10)
	case int32:
		return strconv.FormatInt(int64(m), 10)
	case uint32:
		return strconv.FormatInt(int64(m), 10)
	case int64:
		return strconv.FormatInt(m, 10)
	case uint64:
		return strconv.FormatInt(int64(m), 10)
	case int:
		return strconv.Itoa(m)
	case []byte:
		return ByteSlice2String(m)
	case bool:
		return strconv.FormatBool(m)
	case string:
		return m
	default:
		// nolint:iklint
		panic(fmt.Sprintf("convert %s type to string error ", reflect.TypeOf(m)))
	}

}

func floatToString(f float64) string {
	return strconv.FormatFloat(f, 'f', 3, 64)
}

func getBuffer() (b *bytes.Buffer) {
	bb := bufferPool.Get()
	if bb != nil {
		b, _ = bb.(*bytes.Buffer)
		b.Reset()
		return
	}
	return new(bytes.Buffer)
}

func getMetricName(name string, tags []interface{}) (s string) {
	lenTags := len(tags) / 2
	if lenTags == 0 {
		s = name
		return
	}
	tagPairs := make([][2]string, lenTags)
	for i := 0; i < lenTags; i++ {
		if tag, ok := tags[2*i].(string); ok && tag == TagComment {
			commentAdd(name, tags[2*i+1])
			continue
		}
		if sp, ok := tags[2*i+1].(string); ok && sp == "" {
			continue
		}
		tagPairs[i] = [2]string{builtinToString(tags[2*i]), builtinToString(tags[2*i+1])}
	}
	sort.Sort(kvPairs(tagPairs))
	builder := getBuffer()
	builder.WriteString(name)
	for i, pair := range tagPairs {
		if i == 0 {
			builder.WriteString("|")
		}
		builder.WriteString(pair[0])
		builder.WriteString("=")
		builder.WriteString(pair[1])
		if i != lenTags-1 {
			builder.WriteString(",")
		}
	}
	s = builder.String()
	bufferPool.Put(builder)
	return
}

func mapToString(a map[string]string, excepts ...string) (s string) {
	keys := make([]string, 0, len(a))

loop:
	for k := range a {
		for _, except := range excepts {
			if k == except || k == TagComment {
				continue loop
			}
		}
		keys = append(keys, k)
	}

	lenKeys := len(keys)
	sort.Strings(keys)
	tags := getBuffer()
	for i, k := range keys {
		tags.WriteString(k)
		tags.WriteString("=")
		tags.WriteString(a[k])
		if i != lenKeys-1 {
			tags.WriteString(",")
		}
	}
	s = tags.String()
	bufferPool.Put(tags)
	return
}
