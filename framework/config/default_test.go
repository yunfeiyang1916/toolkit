package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createFileForTest(t *testing.T) *os.File {
	data := []byte(`foo="bar"`)
	path := filepath.Join(os.TempDir(), fmt.Sprintf("file.%d", time.Now().UnixNano()))
	fh, err := os.Create(path)
	if err != nil {
		t.Error(err)
	}
	_, err = fh.Write(data)
	if err != nil {
		t.Error(err)
	}

	return fh
}

func TestLoadWithGoodFile(t *testing.T) {
	fh := createFileForTest(t)
	path := fh.Name()
	defer func() {
		fh.Close()
		os.Remove(path)
	}()

	if err := Default.LoadFile(path); err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}
}

func TestLoadWithInvalidFile(t *testing.T) {
	fh := createFileForTest(t)
	path := fh.Name()
	defer func() {
		fh.Close()
		os.Remove(path)
	}()

	// err := Default.Load(file.NewSource(
	// 	file.WithPath(path),
	// 	file.WithPath("/i/do/not/exists.json"),
	// ))

	err := Files(path, "/i/do/not/exists.json")

	if err == nil {
		t.Fatal("Expected error but none !")
	}
	if !strings.Contains(fmt.Sprintf("%v", err), "/i/do/not/exists.json") {
		t.Fatalf("Expected error to contain the unexisting file but got %v", err)
	}
}

func createFile(fileType string) *os.File {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("file.%d.%s", time.Now().UnixNano(), fileType))
	fh, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
	}
	return fh
}

func TestMultiLoadFile(t *testing.T) {
	f1 := createFile("toml")
	f2 := createFile("json")
	defer func() {
		f1.Close()
		os.Remove(f1.Name())
		f2.Close()
		os.Remove(f2.Name())
	}()

	content := `[log]
	level="debug"
	logpath="./logs/"`

	err := ioutil.WriteFile(f1.Name(), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	c := New()
	err = c.LoadFile(f1.Name())
	err = c.LoadFile(f2.Name())

	// 解析配置
	// err = Files(f1.Name(), f2.Name())
	if err != nil {
		fmt.Println("load files failed")
	}

	assert.Equal(t, c.Get("log", "level").String(""), "debug")
}

func TestIncludeFiles(t *testing.T) {
	f1 := createFile("toml")
	f2 := createFile("toml")
	defer func() {
		f1.Close()
		os.Remove(f1.Name())
		f2.Close()
		os.Remove(f2.Name())
	}()

	content1 := `[log]
	level="debug"
	logpath="./logs/"`

	err := ioutil.WriteFile(f1.Name(), []byte(content1), 0644)
	if err != nil {
		t.Fatal(err)
	}

	content2 := `[server]
  port=9090`

	err = ioutil.WriteFile(f2.Name(), []byte(content2), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 解析配置
	// err = Files(f1.Name(), f2.Name())
	c := New()
	c.LoadFile(f1.Name())
	c.LoadFile(f2.Name())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, c.Get("log", "level").String(""), "debug")
	assert.Equal(t, c.Get("server", "port").Int(0), 9090)
}

func TestConsul(t *testing.T) {
	Consul("/")
	fmt.Println(Default)
}

type Hosts struct {
	Database struct {
		Address string  `json:"address"`
		Port    float64 `json:"port"` // json的整型在golang是float64
	} `json:"database"`
}

type JsonData struct {
	Data Hosts `json:"hosts" toml:"hosts"`
}

type CFG struct {
	LL Log `toml:"log"`
	// Log struct {
	// 	Level string `toml:"level"`
	// 	LogPath string `toml:"path"`
	// } `toml:"log"`
}

type Log struct {
	Level string `toml:"level"`
	Path  string `toml:"path"`
}

func TestDefConfig(t *testing.T) {
	f1 := createFile("toml")
	f2 := createFile("toml")
	defer func() {
		f1.Close()
		os.Remove(f1.Name())
		f2.Close()
		os.Remove(f2.Name())
	}()

	// file 1
	content := `[log]
	level="debug"
	path="./logs/"`

	err := ioutil.WriteFile(f1.Name(), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// file2
	content2 := `[server]
   port=9090`

	err = ioutil.WriteFile(f2.Name(), []byte(content2), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// use default config instance
	// err = Files(f1.Name())
	// if err != nil {
	// 	fmt.Println("load files failed")
	// }
	// assert.Equal(t, Default.Get("log", "level").String("xxx"), "debug")

	// make a new config instance
	cc := New()
	err = cc.LoadFile(f2.Name())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, cc.Get("server", "port").Int(0), 9090)

	// get value by default config instance
	// log := &CFG{}
	// err = Default.Scan(log)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// assert.Equal(t, log.LL.Level, "debug")
}

type DspConfig struct {
	RpcCmd struct {
		Cmd     []string `toml:"cmd"`
		Service string   `toml:"service"`
	} `toml:"rpccmd"`
	Events map[string]map[string]struct {
		Service string `toml:"service"`
		URI     string `toml:"uri"`
	} `toml:"event"`
}

func TestTomlConfig_LoadFile(t *testing.T) {
	content := `[server]
	service_name="link.business.dispnew"
	port = 20019
	remote_servicefind="consul"

[log]
	level="debug"
	logpath="logs"
	rotate="hour"

[data_log]
	path="./logs/statistic.log"
		rotate="hour"

[rpccmd]
	cmd = ["4002", "4004", "4008", "5000"]
	service = "link.business.storerpcnew"

[[server_client]]
	service_name="link.business.storerpcnew"
	proto="rpc"
	balancetype="roundrobin"
	read_timeout=500
	retry_times=3
	endpoints_from="consul"

[[server_client]]
	service_name="musicmove.business.room_live_social"
	proto="http"
	balancetype="roundrobin"
	read_timeout=2500
	retry_times=0
	endpoints_from="consul"


[event.90000]
	[event.90000."c.bq"]
		service = "musicmove.business.room_live_social"
		uri = "/api/social/expression/send"

	[event.90000."c.jr"]
		service = "musicmove.business.action_dispatcher"
		uri = "/dispatcher/DispatcherService/Business"
`

	f1 := createFile("toml")
	defer func() {
		f1.Close()
		os.Remove(f1.Name())
	}()

	err := ioutil.WriteFile(f1.Name(), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// make a new config instance
	cc := New()
	err = cc.LoadFile(f1.Name())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, cc.Get("log", "level").String("xxx"), "debug")

	log := &DspConfig{}
	err = cc.Scan(log)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, log.RpcCmd.Service, "link.business.storerpcnew")
	assert.Equal(t, log.Events["90000"]["c.bq"].URI, "/api/social/expression/send")
}

func TestListen(t *testing.T) {
	data1 := []byte(`{
 "hosts": {
     "database": {
         "address": "10.0.0.1",
         "port": 3306
     },
     "cache": {
         "address": "10.0.0.2",
         "port": 6379
     }
 }
}`)

	f1 := createFile("json")
	defer func() {
		f1.Close()
		os.Remove(f1.Name())
	}()

	err := ioutil.WriteFile(f1.Name(), data1, 0644)
	if err != nil {
		t.Fatal(err)
	}

	cc := New()
	cc.LoadFile(f1.Name())
	j := &JsonData{}
	cc.Scan(j)
	fmt.Println(j)
	r := cc.Listen(j)
	time.Sleep(100 * time.Millisecond)
	port := r.Load().(*JsonData).Data.Database.Port
	assert.Equal(t, 3306, int(port))
}

type Set struct {
	Strv      string            `toml:"strv"`
	Intv      int               `toml:"intv"`
	Boolv     bool              `toml:"boolv"`
	Durationv time.Duration     `toml:"durv"`
	Floatv    float64           `toml:"floatv"`
	StrMap    map[string]string `toml:"strmap"`
	StrSlice  []string          `toml:"strslice"`
}

func TestReadChangeSet(t *testing.T) {
	content := `
strv="b"
intv=1
boolv=true
durv="3s"
floatv=2.5
[strmap]
key1="value1"
key2="value2"
strslice=["a","b","c"]
`

	f1 := createFile("toml")
	defer func() {
		f1.Close()
		os.Remove(f1.Name())
	}()

	err := ioutil.WriteFile(f1.Name(), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cc := New()
	cc.LoadFile(f1.Name())
	set := &Set{}
	cc.Scan(set)
	fmt.Println("bytes:", cc.Bytes())
	fmt.Println("map:", cc.Map())
	cc.Range(func(k string, v interface{}) {
		fmt.Println("key:", k, "value:", v)
	})
	assert.Equal(t, "b", cc.Get("strv").String(""))
	assert.Equal(t, 1, cc.Get("intv").Int(0))
	assert.Equal(t, true, cc.Get("boolv").Bool(false))
	assert.Equal(t, 3*time.Second, cc.Get("durv").Duration(0))
	assert.Equal(t, 2.5, cc.Get("floatv").Float64(0.0))
}

func TestGet(t *testing.T) {
	content := `[limits]
[limits.nick]
  max_len = 20
  type = "string"
[limits.portrait]
  max_len = 100
  type = "string"
[limits.gender]
  type = "number"`

	f1 := createFile("toml")
	defer func() {
		f1.Close()
		os.Remove(f1.Name())
	}()

	err := ioutil.WriteFile(f1.Name(), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cc := New()
	cc.LoadFile(f1.Name())

	v := cc.Get("limits", "nick", "expire")
	fmt.Println(v.Duration(0))

}
