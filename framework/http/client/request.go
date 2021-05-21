package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/yunfeiyang1916/toolkit/framework/utils"
	"github.com/yunfeiyang1916/toolkit/go-tls"
	"github.com/yunfeiyang1916/toolkit/logging"
	"golang.org/x/net/context"
)

type RequestOption struct {
	retryTimes int
	reqTimeout time.Duration // ms
	slowTime   time.Duration // ms
	metricTags map[string]string
}

func (ro *RequestOption) RetryTimes(cnt int) *RequestOption {
	ro.retryTimes = cnt
	return ro
}

func (ro *RequestOption) RequestTimeoutMS(timeout int) *RequestOption {
	ro.reqTimeout = time.Duration(timeout) * time.Millisecond
	return ro
}

func (ro *RequestOption) SlowTimeoutMS(timeout int) *RequestOption {
	ro.slowTime = time.Duration(timeout) * time.Millisecond
	return ro
}

func (ro *RequestOption) MetricTags(t map[string]string) *RequestOption {
	ro.metricTags = t
	return ro
}

type Request struct {
	raw           *http.Request
	ctx           context.Context
	ro            *RequestOption
	url           *url.URL
	path          string
	queryParam    url.Values
	pathParams    map[string]string
	cookies       []*http.Cookie
	header        http.Header
	method        string
	host          string
	form          url.Values
	scheme        string
	buildOnce     bool
	reader        io.Reader
	bigBody       bool
	body          *bytes.Buffer
	simpleBaggage map[string]string
	finalURI      string
}

func NewRequest(ctx ...context.Context) *Request {
	r := &Request{
		ctx: initRequestContext(ctx...),
		ro:  &RequestOption{},
	}
	r.extractBaggage()
	return r
}

func initRequestContext(ctx ...context.Context) context.Context {
	requestCtx := context.Background()
	if len(ctx) > 0 && ctx[0] != nil && ctx[0] != context.TODO() && ctx[0] != context.Background() {
		requestCtx = ctx[0]
	} else {
		if val, ok := tls.GetContext(); ok {
			requestCtx = val
		}
	}
	return requestCtx
}

func (r *Request) WithOption(ro *RequestOption) *Request {
	if ro == nil {
		ro = &RequestOption{}
	}
	r.ro = ro
	return r
}

func (r *Request) WithRequest(req *http.Request) *Request {
	if req == nil {
		return r
	}
	r.raw = cloneRawRequest(req.Context(), req)
	r.method = req.Method
	r.scheme = req.URL.Scheme
	r.header = r.raw.Header
	r.form = r.raw.Form
	r.cookies = r.raw.Cookies()
	r.WithURL(req.URL.String())
	r.WithBody(req.Body)
	return r
}

func (r *Request) WithMethod(method string) *Request {
	r.method = strings.ToUpper(method)
	return r
}

// WithURL() set url.URL{}, after setting, will use this url.URL{} value first and only.
// attention: param uri should not abs url, if uri is abs url, it's scheme and host will be not effect.
func (r *Request) WithURL(uri string) *Request {
	u, err := url.Parse(uri)
	if err != nil {
		logging.GenLogf("invalid uri: %s", uri)
		return r
	}
	r.scheme = u.Scheme
	r.host = u.Host
	r.url = u
	r.path = u.Path
	return r
}

// Deprecated: WithScheme func should not use anymore.
// should config proto type on config: proto="http" or proto="https"
func (r *Request) WithScheme(scheme string) *Request {
	return r
}

func (r *Request) WithPath(path string) *Request {
	if len(path) > 0 {
		r.path = path
	}
	return r
}

func (r *Request) WithStruct(s interface{}) *Request {
	body, _ := enc.Encode(s)
	return r.WithBody(bytes.NewReader(body))
}

func (r *Request) WithBody(body io.Reader) *Request {
	rc, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		rc = ioutil.NopCloser(body)
	}
	if rc == nil {
		return r
	}

	if !r.bigBody {
		b, _ := ioutil.ReadAll(rc)
		r.body = bytes.NewBuffer(b)
		r.reader = bytes.NewBuffer(b)
		return r
	}

	r.reader = rc

	return r
}

func (r *Request) WithBigBody(body io.Reader) *Request {
	r.bigBody = true
	r.WithBody(body)
	return r
}

func (r *Request) WithCookie(ck *http.Cookie) *Request {
	if r.cookies == nil {
		r.cookies = make([]*http.Cookie, 0)
	}
	r.cookies = append(r.cookies, ck)
	return r
}

func (r *Request) WithMultiCookie(cks []*http.Cookie) *Request {
	if r.cookies == nil {
		r.cookies = make([]*http.Cookie, 0)
	}
	r.cookies = append(r.cookies, cks...)
	return r
}

func (r *Request) WithMultiHeader(headers map[string]string) *Request {
	if r.header == nil {
		r.header = http.Header{}
	}
	for k, v := range headers {
		r.header.Set(k, v)
	}
	return r
}

func (r *Request) AddHeader(key, value string) *Request {
	if r.header == nil {
		r.header = http.Header{}
	}
	r.header.Add(key, value)
	return r
}

func (r *Request) DelHeader(key string) *Request {
	if r.header == nil {
		return r
	}
	r.header.Del(key)
	return r
}

// url /v1/user?a=b&c=d
func (r *Request) initQueryParam() {
	if r.queryParam == nil {
		r.queryParam = url.Values{}
	}
}
func (r *Request) WithQueryParam(key, value string) *Request {
	r.initQueryParam()
	r.queryParam.Set(key, value)
	return r
}

func (r *Request) WithQueryParamInt(key string, value int) *Request {
	r.initQueryParam()
	vStr := strconv.Itoa(value)
	r.queryParam.Set(key, vStr)
	return r
}

func (r *Request) WithQueryParamInt64(key string, value int64) *Request {
	r.initQueryParam()
	vStr := strconv.FormatInt(value, 10)
	r.queryParam.Set(key, vStr)
	return r
}

func (r *Request) WithQueryParamUint64(key string, value uint64) *Request {
	r.initQueryParam()
	vStr := strconv.FormatUint(value, 10)
	r.queryParam.Set(key, vStr)
	return r
}

func (r *Request) AddQueryParam(key, value string) *Request {
	r.initQueryParam()
	r.queryParam.Add(key, value)
	return r
}

func (r *Request) AddQueryParamInt(key string, value int) *Request {
	r.initQueryParam()
	vStr := strconv.Itoa(value)
	r.queryParam.Add(key, vStr)
	return r
}

func (r *Request) AddQueryParamInt64(key string, value int64) *Request {
	r.initQueryParam()
	vStr := strconv.FormatInt(value, 10)
	r.queryParam.Add(key, vStr)
	return r
}

func (r *Request) AddQueryParamUint64(key string, value uint64) *Request {
	r.initQueryParam()
	vStr := strconv.FormatUint(value, 10)
	r.queryParam.Add(key, vStr)
	return r
}

func (r *Request) WithMultiQueryParam(params map[string]string) *Request {
	r.initQueryParam()
	for p, v := range params {
		r.queryParam.Set(p, v)
	}
	return r
}

// `application/x-www-form-urlencoded`
func (r *Request) WithFormData(data map[string]string) *Request {
	if r.form == nil {
		r.form = url.Values{}
	}
	for k, v := range data {
		r.form.Set(k, v)
	}
	return r
}

// url params: /v1/users/:userId/:subAccountId/details
func (r *Request) WithPathParams(params map[string]string) *Request {
	if r.pathParams == nil {
		r.pathParams = make(map[string]string)
	}
	for p, v := range params {
		r.pathParams[p] = v
	}
	return r
}

func (r *Request) build() error {
	if r.raw != nil {
		if len(r.host) > 0 {
			r.raw.URL.Host = r.host
		}
	}

	if r.buildOnce {
		return nil
	}
	r.buildOnce = true

	if len(r.method) == 0 {
		r.method = "GET"
	}
	if len(r.scheme) == 0 {
		r.scheme = "http"
	}

	var urlStr string
	if r.url != nil && len(r.url.String()) > 0 {
		r.url.Scheme = r.scheme
		if len(r.host) > 0 {
			r.url.Host = r.host
		}
		r.path = r.url.Path
		urlStr = r.url.String()
		for p, v := range r.pathParams {
			urlStr = strings.Replace(urlStr, ":"+p, url.PathEscape(v), -1)
		}
	} else {
		path := r.path
		if len(path) > 0 {
			for p, v := range r.pathParams {
				path = strings.Replace(path, ":"+p, url.PathEscape(v), -1)
			}
		}
		query := url.Values{}
		for k, v := range r.queryParam {
			for _, iv := range v {
				query.Add(k, iv)
			}
		}
		if len(query) > 0 {
			path = fmt.Sprintf("%s?%s", path, query.Encode())
		}
		urlStr = fmt.Sprintf("%s://%s%s", r.scheme, r.host, path)
	}
	r.finalURI = urlStr
	_, err := url.Parse(urlStr)
	if err != nil {
		return errors.Wrap(err, urlStr)
	}
	nReq, err := http.NewRequest(r.method, urlStr, r.reader)
	if err != nil {
		return errors.Wrap(err, urlStr)
	}
	if len(nReq.Host) == 0 {
		return errors.New("invalid request:http URL no Host")
	}
	r.raw = nReq.WithContext(r.ctx)

	for k, v := range r.header {
		for _, vv := range v {
			r.raw.Header.Add(k, vv)
		}
	}
	for _, k := range r.cookies {
		r.raw.AddCookie(k)
	}
	if len(r.form) > 0 && r.raw.Form == nil {
		r.raw.Form = url.Values{}
	}
	for k, v := range r.form {
		for _, vv := range v {
			r.raw.Form.Add(k, vv)
		}
	}
	return nil
}

// RawRequest() build a http request, after RawRequest(), Request should not be changed by WithXXX func
func (r *Request) RawRequest() *http.Request {
	if err := r.build(); err != nil {
		curTime := time.Now().Format(utils.TimeFormat)
		fmt.Printf("%s,make a new request fail,%v\n", curTime, err)
		return nil
	}
	return r.raw
}

// http clone begin from go1.13
func cloneRawRequest(ctx context.Context, raw *http.Request) *http.Request {
	if raw == nil {
		return nil
	}
	// only clone url
	r2 := raw.WithContext(ctx)

	if raw.Header != nil {
		r2.Header = cloneHeader(raw.Header)
	}
	if raw.Trailer != nil {
		r2.Trailer = cloneHeader(raw.Trailer)
	}
	if s := raw.TransferEncoding; s != nil {
		s2 := make([]string, len(s))
		copy(s2, s)
		r2.TransferEncoding = s
	}
	r2.Form = cloneURLValues(raw.Form)
	r2.PostForm = cloneURLValues(raw.PostForm)
	r2.MultipartForm = cloneMultipartForm(raw.MultipartForm)

	return r2
}

// 将用户自定义信息存储到span baggage item中,在生成http request时执行
func (r *Request) injectBaggage(ctx context.Context) {
	if len(r.simpleBaggage) == 0 {
		r.ctx = ctx
		return
	}
	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		span, ctx = opentracing.StartSpanFromContext(ctx, "inject baggage")
	}
	b, _ := json.Marshal(r.simpleBaggage)
	key := utils.DaeBaggageHeaderPrefix + "baggage"
	r.ctx = opentracing.ContextWithSpan(ctx, span.SetBaggageItem(key, string(b)))
}

// 从已有的ctx中提取span baggage item信息,每个新请求都会执行
func (r *Request) extractBaggage() {
	span := opentracing.SpanFromContext(r.ctx)
	if span == nil {
		return
	}
	if r.simpleBaggage == nil {
		r.simpleBaggage = make(map[string]string)
	}
	val := span.BaggageItem(utils.DaeBaggageHeaderPrefix + "baggage")
	_ = json.Unmarshal([]byte(val), &r.simpleBaggage)
}

// 设置用户自定义信息，用于透传到下游各个节点
func (r *Request) SetBaggage(key, value string) {
	if r.simpleBaggage == nil {
		r.simpleBaggage = make(map[string]string)
	}
	k1 := utils.DaeBaggageHeaderPrefix + key
	r.simpleBaggage[k1] = value
}

// 获取当前请求中已有的baggage信息
func (r *Request) Baggage(key string) string {
	if r.simpleBaggage == nil {
		return ""
	}
	k1 := utils.DaeBaggageHeaderPrefix + key
	return r.simpleBaggage[k1]
}

// 遍历所有已设置的用户自定义baggage信息
func (r *Request) ForeachBaggage(handler func(key, val string) error) error {
	for k, v := range r.simpleBaggage {
		if err := handler(k, v); err != nil {
			return err
		}
	}
	return nil
}
