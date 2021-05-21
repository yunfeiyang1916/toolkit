package tracing

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	_kvPutURI     = "/api/kv/put"
	_traceAPIAddr = "127.0.0.1:5778"
)

func KVPut(body []byte) ([]byte, error) {
	httpclient := http.Client{Timeout: 10 * time.Second}
	url := "http://" + _traceAPIAddr + _kvPutURI
	rsp, err := httpclient.Post(url, "application/json; charset=utf-8", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rsp.Body.Close()
	}()
	respB, _ := ioutil.ReadAll(rsp.Body)
	return respB, nil
}

func InitTraceAPIAddr(addr string) {
	_traceAPIAddr = addr
}
