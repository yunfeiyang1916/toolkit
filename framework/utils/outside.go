package utils

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/go-playground/validator"
	"github.com/gorilla/schema"
	"github.com/pkg/errors"
	"github.com/yunfeiyang1916/toolkit/ecode"
	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/json"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/retry"
	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/sd"
	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
)

const DaeBaggageHeaderPrefix = "daectx-"
const TimeFormat = "2006-01-02 15:04:05.999"

func Register(manager *registry.ServiceManager, appServiceName string, protoType string, tags map[string]string, ip string, port int) (*config.Register, error) {
	var err error
	name := fmt.Sprintf("%s-%s", appServiceName, protoType)
	cfg := config.NewRegister(name, ip, port)
	cfg.ServiceTags = tags
	cfg.TagsWatchPath, err = sd.RegistryKVPath(appServiceName, "/service_tags")
	if err != nil {
		return nil, err
	}
	if manager != nil {
		if err := manager.Register(cfg); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

func MakeAppServiceName(app, name string) string {
	if len(app) == 0 {
		return name
	}
	return app + "." + name
}

func LastError(err error) error {
	switch e := err.(type) {
	case retry.RetryError:
		err = e.Final
	}
	return err
}

func Base64(buf []byte) string {
	// udp 65535byte limit
	limit := 4096
	if len(buf) == 0 || len(buf) > limit {
		return ""
	}
	return base64.StdEncoding.EncodeToString(buf)
}
func DumpRespBody(resp *http.Response) []byte {

	if resp == nil {
		return nil
	}

	// 超过1MB就没必要dump了
	limitBody := 1*1024*1024 + 1
	pieceBuf := bytes.NewBuffer(nil)
	// piece reader
	lr := io.LimitReader(resp.Body, int64(limitBody))
	pieceBuf.ReadFrom(lr)
	if pieceBuf.Len() == limitBody {
		// 还原body, rebuild body reader
		mr := io.MultiReader(pieceBuf, resp.Body)
		resp.Body = ioutil.NopCloser(mr)
		return nil
	}

	buf, err := ioutil.ReadAll(pieceBuf)
	if err != nil || len(buf) == 0 {
		return nil
	}
	resp.Body = ioutil.NopCloser(bytes.NewReader(buf))
	return buf
}

var (
	URIDecoder = schema.NewDecoder()
	Valid      = validator.New()
)

func init() {
	URIDecoder.IgnoreUnknownKeys(true)
}

func Bind(raw *http.Request, model interface{}, obj ...interface{}) error {
	bodyBytes := make([]byte, 0)
	if raw.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(raw.Body)
		defer func() {
			raw.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		}()
	}
	method := raw.Method
	reqUrl := raw.URL.String()
	switch method {
	case "GET":
		dErr := URIDecoder.Decode(model, raw.URL.Query())
		vErr := Valid.Struct(model)
		if dErr != nil || vErr != nil {
			return errors.Wrapf(ecode.ParamErr, "bind failed,reqUrl(%s),body(%s),err(%v,%v)", reqUrl, bodyBytes, dErr, vErr)
		}
	case "POST":
		// parse query
		err := URIDecoder.Decode(model, raw.URL.Query())
		if len(bodyBytes) == 0 { // 没有body直接校验结构
			if err != nil {
				return errors.Wrapf(ecode.ParamErr, "bind query failed,reqUrl(%s),body(%s),err(%v)", reqUrl, bodyBytes, err)
			}
			if vErr := Valid.Struct(model); vErr != nil {
				return errors.Wrapf(ecode.ParamErr, "struct invalid,reqUrl(%s),body(%s),err(%v)", reqUrl, bodyBytes, vErr)
			}
			return nil
		}
		// parse body, 会覆盖query中的同名参数
		dErr := json.NewEncoder().Decode(bodyBytes, &model)
		vErr := Valid.Struct(model)
		if dErr != nil || vErr != nil {
			return errors.Wrapf(ecode.ParamErr, "bind body failed,reqUrl(%s),body(%s),err(%v,%v)", reqUrl, bodyBytes, dErr, vErr)
		}
	default:
	}
	if len(obj) > 0 {
		dErr := URIDecoder.Decode(obj[0], raw.URL.Query())
		vErr := Valid.Struct(obj[0])
		if dErr != nil || vErr != nil {
			return errors.Wrapf(ecode.ParamErr, "bind failed,reqUrl(%s),body(%s),err(%v,%v)", reqUrl, bodyBytes, dErr, vErr)
		}
	}
	return nil
}

type WrapResp struct {
	Code int         `json:"dm_error"`
	Msg  string      `json:"error_msg"`
	Data interface{} `json:"data"`
}

func NewWrapResp(data interface{}, err error) WrapResp {
	e := ecode.Cause(err)
	return WrapResp{
		Code: e.Code(),
		Msg:  e.Message(),
		Data: data,
	}
}

func LenSyncMap(m *sync.Map) int {
	length := 0
	m.Range(func(key, value interface{}) bool {
		length++
		return true
	})
	return length
}
