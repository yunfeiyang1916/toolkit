package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/yunfeiyang1916/toolkit/framework/internal/core"
	"github.com/yunfeiyang1916/toolkit/framework/utils"
	"golang.org/x/net/context"
)

type internalContext struct{}

var iCtxKey = internalContext{}

type Context struct {
	Request      *http.Request
	Response     Responser
	Params       Params
	Path         string // raw match path
	Peer         string // 包含app_name的上游service_name
	Namespace    string
	Ctx          context.Context // for trace or others store
	core         core.Core       // a control flow
	w            *responseWriter
	busiCode     int32
	loggingExtra map[string]interface{}
	// use url.ParseQuery cached the param query result from c.Request.URL.Query()
	queryCache    url.Values
	bodyBuff      *bytes.Buffer
	printRespBody bool
	printReqBody  bool
	srv           *server
	traceId       string
	startTime     time.Time
	Keys          map[string]interface{}
	simpleBaggage map[string]string
}

func (c *Context) reset() {
	c.Request = nil
	c.Response = nil
	c.Params = c.Params[0:0]
	c.Path = ""
	c.Peer = ""
	c.Ctx = nil
	c.Namespace = ""
	c.core = nil
	c.w = &responseWriter{}
	c.busiCode = 0
	c.loggingExtra = nil
	c.queryCache = nil
	c.bodyBuff = bytes.NewBuffer(nil)
	c.printRespBody = true
	c.printReqBody = true
	c.traceId = ""
	c.Keys = nil
	c.simpleBaggage = nil
}

func (c *Context) chain() []core.Plugin {
	t := c.srv.trees
	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method != c.Request.Method {
			continue
		}
		root := t[i].root
		// plugin, urlparam, found, matchPath expression
		plugins, params, _, mpath := root.getValue(c.Request.URL.Path, c.Params, false)
		if plugins != nil {
			c.Params = params
			c.Path = mpath
			return plugins
		}
		break
	}
	return nil
}

func (c *Context) writeHeaderOnce() {
	c.Response.writeHeaderOnce()
}

func (c *Context) Next() {
	c.core.Next(c.Ctx)
}

func (c *Context) Abort() {
	c.core.Abort()
}

func (c *Context) AbortErr(err error) {
	c.core.AbortErr(err)
}

func (c *Context) Err() error {
	return c.core.Err()
}

func (c *Context) TraceID() string {
	return c.traceId
}

func (c *Context) SetBusiCode(code int32) {
	atomic.StoreInt32(&c.busiCode, code)
}

func (c *Context) BusiCode() int32 {
	return atomic.LoadInt32(&c.busiCode)
}

func (c *Context) LoggingExtra(vals ...interface{}) {
	if c.loggingExtra == nil {
		c.loggingExtra = map[string]interface{}{}
	}
	if len(vals)%2 != 0 {
		vals = append(vals, "<kv not match>")
	}
	size := len(vals)
	for i := 0; i < size; i += 2 {
		key := fmt.Sprintf("%v", vals[i])
		c.loggingExtra[key] = vals[i+1]
	}
}

func (c *Context) Bind(r *http.Request, model interface{}, atom ...interface{}) error {
	return utils.Bind(r, model, atom...)
}

// write response, error include business code and error msg
func (c *Context) JSON(data interface{}, err error) {
	c.Response.WriteHeader(c.Response.Status())
	w := utils.NewWrapResp(data, err)
	c.SetBusiCode(int32(w.Code))
	_, _ = c.Response.WriteJSON(w)
}

// wrap on JSON
func (c *Context) JSONAbort(data interface{}, err error) {
	c.JSON(data, err)
	c.Abort()
}

// JSONOrError will handle error and write either JSON or error to response.
// good example:
// ```
//     resp, err := svc.MessageService()
//     c.JSONOrError(resp, err)
// ```
// bad example:
// ```
//     resp, err := svc.MessageService()
//     c.JSON(resp, err)
// ```
func (c *Context) JSONOrError(data interface{}, err error) {
	if err != nil {
		c.JSONAbort(nil, err)
		return
	}
	c.JSON(data, nil)
}

func (c *Context) DefaultQuery(key, defaultValue string) string {
	if value, ok := c.GetQuery(key); ok {
		return value
	}
	return defaultValue
}

// Query returns the keyed url query value if it exists,
// otherwise it returns an empty string `("")`.
// It is shortcut for `c.Request.URL.Query().Get(key)`
//     GET /path?id=1234&name=Manu&value=
// 	   c.Query("id") == "1234"
// 	   c.Query("name") == "Manu"
// 	   c.Query("value") == ""
// 	   c.Query("wtf") == ""
func (c *Context) Query(key string) string {
	value, _ := c.GetQuery(key)
	return value
}

func (c *Context) QueryInt(key string) int {
	i, _ := strconv.Atoi(c.Query(key))
	return i
}

func (c *Context) QueryInt64(key string) int64 {
	i, _ := strconv.ParseInt(c.Query(key), 10, 64)
	return i
}

func (c *Context) GetQuery(key string) (string, bool) {
	if values, ok := c.GetQueryArray(key); ok {
		return values[0], ok
	}
	return "", false
}

func (c *Context) QueryArray(key string) []string {
	values, _ := c.GetQueryArray(key)
	return values
}

func (c *Context) GetQueryArray(key string) ([]string, bool) {
	if c.queryCache == nil {
		c.queryCache, _ = url.ParseQuery(c.Request.URL.RawQuery)
	}

	if values, ok := c.queryCache[key]; ok && len(values) > 0 {
		return values, true
	}
	return []string{}, false
}

// write response, data include error info
func (c *Context) Raw(data interface{}, code int32) {
	c.SetBusiCode(code)
	_, _ = c.Response.WriteJSON(data)
}

func (c *Context) Set(key string, value interface{}) {
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[key] = value
}

func (c *Context) Get(key string) (value interface{}, exists bool) {
	value, exists = c.Keys[key]
	return
}

func (c *Context) MustGet(key string) interface{} {
	if value, exists := c.Get(key); exists {
		return value
	}
	panic("Key \"" + key + "\" does not exist")
}

func (c *Context) GetString(key string) (s string) {
	if val, ok := c.Get(key); ok && val != nil {
		s, _ = val.(string)
	}
	return
}

func (c *Context) GetBool(key string) (b bool) {
	if val, ok := c.Get(key); ok && val != nil {
		b, _ = val.(bool)
	}
	return
}

func (c *Context) GetInt(key string) (i int) {
	if val, ok := c.Get(key); ok && val != nil {
		i, _ = val.(int)
	}
	return
}

func (c *Context) GetInt64(key string) (i64 int64) {
	if val, ok := c.Get(key); ok && val != nil {
		i64, _ = val.(int64)
	}
	return
}

func (c *Context) GetFloat64(key string) (f64 float64) {
	if val, ok := c.Get(key); ok && val != nil {
		f64, _ = val.(float64)
	}
	return
}

func (c *Context) GetTime(key string) (t time.Time) {
	if val, ok := c.Get(key); ok && val != nil {
		t, _ = val.(time.Time)
	}
	return
}

func (c *Context) GetDuration(key string) (d time.Duration) {
	if val, ok := c.Get(key); ok && val != nil {
		d, _ = val.(time.Duration)
	}
	return
}

func (c *Context) GetStringSlice(key string) (ss []string) {
	if val, ok := c.Get(key); ok && val != nil {
		ss, _ = val.([]string)
	}
	return
}

func (c *Context) GetStringMap(key string) (sm map[string]interface{}) {
	if val, ok := c.Get(key); ok && val != nil {
		sm, _ = val.(map[string]interface{})
	}
	return
}

func (c *Context) GetStringMapString(key string) (sms map[string]string) {
	if val, ok := c.Get(key); ok && val != nil {
		sms, _ = val.(map[string]string)
	}
	return
}

func (c *Context) GetStringMapStringSlice(key string) (smss map[string][]string) {
	if val, ok := c.Get(key); ok && val != nil {
		smss, _ = val.(map[string][]string)
	}
	return
}

// 从ctx中提取用户自定义的baggage信息到本地存储,在server接收到http请求时执行
func (c *Context) extractBaggage() {
	span := opentracing.SpanFromContext(c.Ctx)
	if span == nil {
		return
	}
	if c.simpleBaggage == nil {
		c.simpleBaggage = make(map[string]string)
	}
	val := span.BaggageItem(utils.DaeBaggageHeaderPrefix + "baggage")
	_ = json.Unmarshal([]byte(val), &c.simpleBaggage)
}

// 设置用户自定义信息到span baggage item中
func (c *Context) SetBaggage(key, value string) context.Context {
	if c.simpleBaggage == nil {
		c.simpleBaggage = make(map[string]string)
	}
	k1 := utils.DaeBaggageHeaderPrefix + key
	c.simpleBaggage[k1] = value
	span := opentracing.SpanFromContext(c.Ctx)
	b, _ := json.Marshal(c.simpleBaggage)
	span.SetBaggageItem(utils.DaeBaggageHeaderPrefix+"baggage", string(b))
	return opentracing.ContextWithSpan(c.Ctx, span)
}

// 获取已有的用户自定义baggage信息
func (c *Context) Baggage(key string) string {
	if c.simpleBaggage == nil {
		return ""
	}
	k1 := utils.DaeBaggageHeaderPrefix + key
	return c.simpleBaggage[k1]
}

// 遍历处理用户自定义的baggage信息
func (c *Context) ForeachBaggage(handler func(key, val string) error) error {
	for k, v := range c.simpleBaggage {
		if err := handler(k, v); err != nil {
			return err
		}
	}
	return nil
}
