package utils

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Model struct {
	ID   string `schema:"id"`
	Name string `schema:"name"`
}

type Atom struct {
	Group string `schema:"group"`
}

func TestBind(t *testing.T) {
	req1, _ := http.NewRequest("GET", "/abc/login?id=111111&name=jake", nil)
	m1 := new(Model)
	err := Bind(req1, m1)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "111111", m1.ID)
	assert.Equal(t, "jake", m1.Name)

	body := []byte(`{
"id":"2222",
"name":"tomy"
}`)

	req2, _ := http.NewRequest("POST", "/abc/login", bytes.NewBuffer(body))
	m2 := new(Model)
	err2 := Bind(req2, m2)
	if err2 != nil {
		panic(err2)
	}
	assert.Equal(t, "2222", m2.ID)
	assert.Equal(t, "tomy", m2.Name)

	body3 := []byte(`{
"id":"2222"
}`)
	req3, _ := http.NewRequest("POST", "/abc/login?id=111111&name=jake", bytes.NewBuffer(body3))
	m3 := new(Model)
	err3 := Bind(req3, m3)
	if err3 != nil {
		panic(err3)
	}
	assert.Equal(t, "2222", m3.ID)
	assert.Equal(t, "jake", m3.Name)

	body4 := []byte(`{
"name":"kaka"
}`)
	req4, _ := http.NewRequest("POST", "/abc/login?id=1111&name=jake", bytes.NewBuffer(body4))
	m4 := new(Model)
	err4 := Bind(req4, m4)
	if err4 != nil {
		panic(err4)
	}
	assert.Equal(t, "1111", m4.ID)
	assert.Equal(t, "kaka", m4.Name)

	body5 := []byte(`{
"id":"5555",
"name":"hanny"
}`)
	req5, _ := http.NewRequest("POST", "/abc/login?id=1111&name=jake&group=usa", bytes.NewBuffer(body5))
	m5 := new(Model)
	a1 := new(Atom)
	err5 := Bind(req5, m5, a1)
	if err5 != nil {
		panic(err5)
	}
	assert.Equal(t, "1111", m4.ID)
	assert.Equal(t, "kaka", m4.Name)
	assert.Equal(t, "usa", a1.Group)
}

func TestLenSyncMap(t *testing.T) {
	m := sync.Map{}
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			m.Store(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
		}
	}(&wg)

	//wg.Add(1)
	//go func() {
	//	defer wg.Done()
	//	for {
	//		l := LenSyncMap(m)
	//		fmt.Printf("sync map length:%d\n", l)
	//		if l == 100 {
	//			return
	//		}
	//	}
	//}()

	wg.Wait()
	l := LenSyncMap(&m)
	fmt.Printf("sync map length:%d\n", l)
}
