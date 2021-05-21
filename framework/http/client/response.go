package client

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/opentracing/opentracing-go"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/json"
)

type Response struct {
	err    error
	code   int            // http status code, value from resp.StatusCode
	rsp    *http.Response // raw http.Response
	req    *http.Request  // raw http request
	buffer *bytes.Buffer  // rsp body buffer copy
	cancel context.CancelFunc
	span   opentracing.Span
}

// Response should be close by caller
func BuildResp(req *http.Request, resp *http.Response) (*Response, error) {
	res := &Response{
		rsp:    resp,
		req:    req,
		buffer: bytes.NewBuffer(nil),
		code:   -1,
	}
	if resp != nil {
		res.code = resp.StatusCode
		resp.Body = &Body{cancel: res.cancel, rc: res.rsp.Body, span: res.span}
	}

	return res, res.err
}

func (r *Response) Error() error {
	return r.err
}

func (r *Response) Code() int {
	return r.code
}

func (r *Response) RawRequest() *http.Request {
	return r.req
}

func (r *Response) RawResponse() *http.Response {
	if r.buffer.Len() > 0 {
		r.rsp.Body = ioutil.NopCloser(bytes.NewBuffer(r.buffer.Bytes()))
	}
	return r.rsp
}

func (r *Response) setCancel(fn context.CancelFunc) {
	r.cancel = fn
}

func (r *Response) setSpan(span opentracing.Span) {
	r.span = span
}

type Body struct {
	cancel context.CancelFunc
	rc     io.ReadCloser
	span   opentracing.Span
}

func (b *Body) Close() error {
	err := b.rc.Close()

	if b.cancel != nil {
		b.cancel()
	}
	if b.span != nil {
		b.span.LogFields(opentracinglog.String("event", "ClosedBody"))
		b.span.Finish()
	}
	return err
}

func (b *Body) Read(data []byte) (int, error) {
	return b.rc.Read(data)
}

func (r *Response) Body() *Body {
	if r.rsp == nil {
		return nil
	}
	return &Body{cancel: r.cancel, rc: r.rsp.Body, span: r.span}
}

func (r *Response) Bytes() []byte {
	r.makeRspByteBuffer()
	return r.buffer.Bytes()
}

func (r *Response) String() string {
	r.makeRspByteBuffer()
	return r.buffer.String()
}

var enc = json.NewEncoder()

func (r *Response) JSON(obj interface{}) error {
	return enc.Decode(r.Bytes(), obj)
}

func (r *Response) Save(fileName string) error {
	fd, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer fd.Close()

	if r.rsp == nil {
		return nil
	}

	defer r.Body().Close()

	_, err = io.Copy(fd, r.rsp.Body)
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (r *Response) makeRspByteBuffer() {
	if r.buffer.Len() != 0 || r.rsp == nil {
		return
	}
	defer r.Body().Close()

	_, err := io.Copy(r.buffer, r.rsp.Body)
	if err != nil {
		if r.err != nil {
			r.err = errors.Wrap(r.err, err.Error())
		} else {
			r.err = err
		}
	}
}

func (r *Response) GetHeader(key string) string {
	return r.rsp.Header.Get(key)
}
