package client

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"golang.org/x/net/context"
)

type testKey struct {
}

func TestNewRequest(t *testing.T) {
	r := NewRequest()
	fmt.Printf("---1---:%+v\n", *r)
	fmt.Printf("---1---: context: %p, %v\n", r.ctx, r.ctx)

	buf := bytes.NewBufferString("xxxxxx")
	req, _ := http.NewRequest("GET", "/a/b?c=d&e=f", ioutil.NopCloser(buf))
	r.WithRequest(req)
	// fmt.Printf("---2---:%+v\n url:%+v\n body: %+v\n", *r, r.url, r.body)

	ro := &RequestOption{}
	ro.RetryTimes(2)
	ro.RequestTimeoutMS(100)
	ro.SlowTimeoutMS(300)

	fmt.Printf("---2---: context: %p, %v\n", r.ctx, r.ctx)

	ctx := context.WithValue(r.ctx, testKey{}, "aaaa")
	r2 := NewRequest(ctx)
	r2.WithMethod("POST").WithURL("https://127.0.0.1:8080/a/b/c?aa=1&bb=2&cc=3")

	fmt.Printf("---3---:%+v\n", r2.RawRequest())

	r2.AddHeader("aaa", "bbb")
	fmt.Printf("---4---:%+v\n", r2.RawRequest())

	s1 := "http://lcoalhost:8080/user?id=1"
	u1, _ := url.Parse(s1)
	fmt.Println(u1.IsAbs())

	s2 := "lcoalhost:8080/user?id=1"
	u2, _ := url.Parse(s2)
	fmt.Println(u2.IsAbs())

	s3 := "/user?id=1"
	u3, _ := url.Parse(s3)
	fmt.Println(u3.IsAbs())

}
