package server

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"sync"

	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/json"
)

type Responser interface {
	http.ResponseWriter
	Size() int
	Status() int
	Writer() http.ResponseWriter
	WriteString(string) (int, error)
	WriteJSON(interface{}) (int, error)
	ByteBody() []byte
	StringBody() string
	writeHeaderOnce()
}

const (
	noWritten = 0
)

var jsonContentType = []string{"application/json; charset=utf-8"}

var _ Responser = &responseWriter{}

type responseWriter struct {
	http.ResponseWriter
	size   int
	status int
	limit  int
	buff   bytes.Buffer // only cached limit size
	header http.Header
	once   sync.Once
}

func (w *responseWriter) reset(writer http.ResponseWriter, limit int) {
	w.ResponseWriter = writer
	w.size = noWritten
	w.status = http.StatusOK
	w.limit = limit
}

func (w *responseWriter) Size() int {
	return w.size
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) Writer() http.ResponseWriter {
	return w
}

func (w *responseWriter) ByteBody() []byte {
	return w.buff.Bytes()
}

func (w *responseWriter) StringBody() string {
	return w.buff.String()
}

// override
func (w *responseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}

// override
func (w *responseWriter) Write(data []byte) (int, error) {
	return w.write(data)
}

// override
func (w *responseWriter) WriteHeader(statusCode int) {
	if statusCode > 0 && w.status != statusCode {
		w.status = statusCode
	}
}

func (w *responseWriter) WriteString(s string) (int, error) {
	return w.write([]byte(s))
}

func (w *responseWriter) WriteJSON(data interface{}) (n int, err error) {
	b, err := json.NewEncoder().Encode(data)
	if err != nil {
		return
	}
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = jsonContentType
	}
	n, err = w.Write(b)
	return
}

func (w *responseWriter) writeHeaderOnce() {
	w.once.Do(func() {
		for k, v := range w.header {
			for _, vv := range v {
				w.ResponseWriter.Header().Add(k, vv)
			}
		}
		// header写入顺序参考WriteHeader(statusCode int)函数
		w.ResponseWriter.WriteHeader(w.status)
	})
}

func (w *responseWriter) write(data []byte) (n int, err error) {
	w.writeHeaderOnce()
	if w.buff.Len() < w.limit {
		if len(data) < w.limit {
			w.buff.Write(data)
		} else {
			w.buff.Write(data[:w.limit])
		}
	}
	n, err = w.ResponseWriter.Write(data)
	w.size += n
	return
}

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

func (w *responseWriter) Flush() {
	w.writeHeaderOnce()
	w.ResponseWriter.(http.Flusher).Flush()
}
