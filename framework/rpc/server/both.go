package server

import (
	"fmt"
	"sync"

	"github.com/yunfeiyang1916/toolkit/framework/config/encoder/json"

	"github.com/yunfeiyang1916/toolkit/framework/internal/kit/tracing"
	"github.com/yunfeiyang1916/toolkit/logging"

	"github.com/pkg/errors"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/codec"
)

type Both struct {
	router      *router
	binary      Server
	http        Server
	exitC       chan error
	startC      chan error
	serviceName string
}

func BothServer(serviceName string, port int, options ...Option) Server {
	b := &Both{serviceName: serviceName}
	b.router = newRouter()
	b.exitC = make(chan error, 2)
	b.startC = make(chan error, 2)

	ops1 := append(
		options,
		Address(fmt.Sprintf(":%d", port)),
		Codec(codec.NewProtoCodec()),
	)
	b.binary = BinaryServer(ops1...)

	ops2 := append(
		options,
		Address(fmt.Sprintf(":%d", port+1)),
		Codec(codec.NewJSONCodec()),
	)
	b.http = HTTPServer(ops2...)
	b.Use(RatelimitPlugin)
	b.Use(BreakerPlugin)
	return b
}

func (b *Both) NewHandler(handler interface{}, opts ...HandlerOption) Handler {
	return b.router.NewHandler(handler, opts...)
}

func (b *Both) Handle(h Handler) error {
	if err := b.binary.Handle(h); err != nil {
		return err
	}
	if err := b.http.Handle(h); err != nil {
		return err
	}
	return nil
}

func (b *Both) Use(list ...Plugin) Server {
	b.binary.Use(list...)
	b.http.Use(list...)
	return b
}

func (b *Both) Start() error {
	var err error
	eChan := make(chan error, 2)
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		if e := b.binary.Start(); e != nil {
			eChan <- e
		}
	}()

	go func() {
		defer wg.Done()
		if e := b.http.Start(); e != nil {
			eChan <- e
		}
	}()

	go func() {
		for e := range eChan {
			if err == nil {
				err = e
			} else {
				err = errors.Wrap(err, e.Error())
			}
		}
	}()

	b.uploadServerPath()

	wg.Wait()
	close(eChan)
	return err
}

func (b *Both) Stop() error {
	err1 := b.binary.Stop()
	err2 := b.http.Stop()
	if err1 == nil && err2 == nil {
		return nil
	}

	if err1 != nil && err2 != nil {
		return errors.Wrap(err1, err2.Error())
	}

	if err1 == nil {
		return err2
	}
	return err1
}

func (b *Both) uploadServerPath() {
	body := map[string]interface{}{}
	body["type"] = 1
	body["service"] = b.serviceName
	body["resource_list"] = b.GetPaths()
	bodyB, _ := json.NewEncoder().Encode(body)
	respB, err := tracing.KVPut(bodyB)
	if err != nil {
		return
	}
	logging.GenLogf("sync rpc server path list to consul response:%q", respB)
}

func (b *Both) GetPaths() []string {
	return b.binary.GetPaths()
}
