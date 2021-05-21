package toml

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yunfeiyang1916/toolkit/framework/config/source"
)

var content = `[server]
	service_name="test"
	port = 12345
	switch=true
	thold=0.9

[log]
	level="debug"
	durate="30s"

[rpccmd]
	cmd = ["4002", "4004", "4008", "5000"]

[[server_client]]
	service_name="aaa"

[[server_client]]
	service_name="bbb"


[event.90000]
	[event.90000."c.bq"]
		uri = "/a/b/c"

	[event.90000."c.jr"]
		uri = "/1/2/3"
`

var data = []byte(content)
var values, _ = newValues(&source.ChangeSet{
	Data: data,
})

func TestTomlValue_String(t *testing.T) {
	testData := []struct {
		path  []string
		value interface{}
	}{
		{
			[]string{"server", "service_name"},
			"test",
		},
		{
			[]string{"log", "level"},
			"debug",
		},
		{
			[]string{"event", "90000", "c.bq", "uri"},
			"/a/b/c",
		},
	}

	for _, test := range testData {
		if v := values.Get(test.path...).String("xxxx"); v != test.value {
			t.Fatalf("Expected %s got %s for path %v", test.value, v, test.path)
		}
	}
}

func TestTomlValue_Bool(t *testing.T) {
	testData := []struct {
		path  []string
		value interface{}
	}{
		{
			[]string{"server", "switch"},
			true,
		},
	}

	for _, test := range testData {
		if v := values.Get(test.path...).Bool(false); v != test.value {
			t.Fatalf("Expected %v got %v for path %v", test.value, v, test.path)
		}
	}
}

func TestTomlValue_Int(t *testing.T) {
	testData := []struct {
		path  []string
		value interface{}
	}{
		{
			[]string{"server", "port"},
			12345,
		},
	}

	for _, test := range testData {
		if v := values.Get(test.path...).Int(0); v != test.value {
			t.Fatalf("Expected %v got %v for path %v", test.value, v, test.path)
		}
	}
}

func TestTomlValue_Float64(t *testing.T) {
	testData := []struct {
		path  []string
		value interface{}
	}{
		{
			[]string{"server", "thold"},
			0.9,
		},
	}

	for _, test := range testData {
		if v := values.Get(test.path...).Float64(0.0); v != test.value {
			t.Fatalf("Expected %v got %v for path %v", test.value, v, test.path)
		}
	}
}

func TestTomlValue_Duration(t *testing.T) {
	testData := []struct {
		path  []string
		value time.Duration
	}{
		{
			[]string{"log", "durate"},
			30 * time.Second,
		},
	}

	for _, test := range testData {
		if v := values.Get(test.path...).Duration(0); v != test.value {
			t.Fatalf("Expected %v got %v for path %v", test.value, v, test.path)
		}
	}
}

func TestTomlValue_StringSlice(t *testing.T) {
	testData := []struct {
		path  []string
		value []string
	}{
		{
			[]string{"rpccmd", "cmd"},
			[]string{"4002", "4004", "4008", "5000"},
		},
	}

	for _, test := range testData {
		v := values.Get(test.path...).StringSlice(nil)
		for i, vv := range v {
			if vv != test.value[i] {
				t.Fatalf("Expected %v got %v for path %v", test.value[i], vv, test.path)
			}
		}
	}
}

func TestTomlValue_StringMap(t *testing.T) {
	testData := []struct {
		path  []string
		value map[string]string
	}{
		{
			[]string{"event", "90000", "c.bq"},
			map[string]string{"uri": "/a/b/c"},
		},
	}

	for _, test := range testData {
		v := values.Get(test.path...).StringMap(nil)
		fmt.Println(v)
		for kk, vv := range v {
			if test.value[kk] != vv {
				t.Fatalf("Expected %v got %v for path %v", test.value[kk], vv, test.path)
			}
		}
	}
}

type DspConfig struct {
	Events map[string]map[string]struct {
		Service string `toml:"service"`
		URI     string `toml:"uri"`
	} `toml:"event"`
}

type Item struct {
	URI string `toml:"uri"`
}

func TestTomlValues_Scan(t *testing.T) {
	d := &DspConfig{}
	err := values.Scan(d)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, d.Events["90000"]["c.bq"].URI, "/a/b/c")

	tt := &Item{}
	values.Get("event", "90000", "c.bq").Scan(tt)
	assert.Equal(t, tt.URI, "/a/b/c")
}
