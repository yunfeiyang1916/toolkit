package memory

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/json"
	"github.com/yunfeiyang1916/toolkit/framework/config/reader/toml"
	"github.com/yunfeiyang1916/toolkit/framework/config/source"
	"github.com/yunfeiyang1916/toolkit/framework/config/source/file"
)

type Hosts struct {
	Database struct {
		Address string  `json:"address"`
		Port    float64 `json:"port"` //json的整型在golang是float64
	} `json:"database"`
}

type JsonData struct {
	Data Hosts `json:"hosts" toml:"hosts"`
}

func createFile(fileType string) *os.File {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("file.%d.%s", time.Now().UnixNano(), fileType))
	fh, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
	}
	return fh
}

func TestMemory_Listen(t *testing.T) {

	data1 := []byte(`{
 "hosts": {
     "database": {
         "address": "10.0.0.1",
         "port": 3306
     }
 }
}`)
	_ = data1

	data2 := []byte(`{
 "hosts": {
     "database": {
         "address": "101.0.0.2",
         "port": 3306
     }
 }
}`)

	_ = data2

	f1 := createFile("json")
	defer func() {
		f1.Close()
		os.Remove(f1.Name())
	}()

	err := ioutil.WriteFile(f1.Name(), []byte(data2), 0644)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		tick := time.NewTicker(500 * time.Millisecond)
		change := make(chan bool, 1)
		value := true
		for {
			select {
			case <-tick.C:
				change <- value
			case v := <-change:
				var data []byte
				if v {
					data = data1
					value = false //下次用data2
				} else {
					data = data2
					value = true //下次用data1
				}
				err := ioutil.WriteFile(f1.Name(), []byte(data), 0644)
				if err != nil {
					t.Fatal(err)
				}
				fmt.Println("file had lisChan...")
			}
		}
	}()

	time.Sleep(1 * time.Second)

	s := file.NewSource(
		file.WithPath(f1.Name()),
		source.WithEncoder(json.NewEncoder()))

	m := NewLoader()
	m.Load(s)
	m.Sync()
	sp, _ := m.Snapshot()
	//fmt.Println(sp.ChangeSet.Data)

	w, err := m.Watch("hosts", "database")
	if err != nil {
		panic(err)
	}
	go func() {
		sp1, err := w.Next()
		if err == nil {
			fmt.Println(">>>>>>", sp1.ChangeSet)
		}

	}()

	time.Sleep(5 * time.Second)
	w.Stop()

	reader := toml.NewReader()
	v, _ := reader.Values(sp.ChangeSet)
	j := &JsonData{}
	v.Scan(j)
	fmt.Println(j)

	r := m.Listen(j)
	time.Sleep(100 * time.Millisecond)
	port := r.Load().(*JsonData).Data.Database.Port
	m.Close()

	assert.Equal(t, int(port), 3306)

}
