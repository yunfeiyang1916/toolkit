package client

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/client/mocks"
	"github.com/yunfeiyang1916/toolkit/framework/rpc/codec"
	"github.com/yunfeiyang1916/toolkit/go-upstream/config"
	"github.com/yunfeiyang1916/toolkit/go-upstream/upstream"
	"golang.org/x/net/context"
)

var jsonc = codec.NewJSONCodec()
var protocodec = codec.NewProtoCodec()

func dialerTestOpt(d dialer) Option {
	return func(o *Options) {
		o.dialer = d
	}
}

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestClientGeneral(t *testing.T) {
	assert := assert.New(t)

	clusterName := "test_client"

	config := config.NewCluster()
	config.Name = clusterName
	config.StaticEndpoints = "127.0.0.1:7832"

	manager := upstream.NewClusterManager()
	assert.Nil(manager.InitService(config))

	var sockettest = map[string]struct {
		body []byte
		err  error
	}{
		"error_endpoint": {
			err: errors.New("just an error"),
		},
	}

	mockdialer := new(mockedDialer)
	mockdialer.On("Dial", mock.Anything).Return(func(string) socket {
		socket := new(mocks.Socket)
		socket.On("Close").Return(nil)
		socket.On("Call", mock.Anything, mock.Anything, mock.Anything).Return(
			func(endpoint string, header map[string]string, body []byte) []byte {
				if _, ok := sockettest[endpoint]; !ok {
					// echo
					return body
				}
				return sockettest[endpoint].body
			},
			func(endpoint string, header map[string]string, body []byte) error {
				if _, ok := sockettest[endpoint]; !ok {
					return nil
				}
				return sockettest[endpoint].err
			})
		return socket
	}, nil)

	client := SFactory(
		Cluster(manager.Cluster(clusterName)),
		Codec(jsonc),
		dialerTestOpt(mockdialer), // use mock dialer,
	)
	assert.NotNil(client)

	tests := map[string]struct {
		request  interface{}
		response interface{}
		expect   interface{}
		err      error
		endpoint string
	}{
		"echo_success_1": {
			endpoint: "endpoint",
			request:  Person{Name: "Sam", Age: 10},
			response: &Person{},
			expect:   &Person{Name: "Sam", Age: 10},
		},
		"error_1": {
			endpoint: "error_endpoint",
			request:  Person{Name: "Sam", Age: 10},
			response: &Person{},
			expect:   &Person{},
			err:      sockettest["error_endpoint"].err,
		},
	}

	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			err := client.Client(test.endpoint).Invoke(context.TODO(), test.request, test.response)
			if err != nil {
				assert.EqualError(err, test.err.Error())
			} else {
				assert.Equal(test.err, err)
			}
			assert.Equal(test.expect, test.response)
		})
	}
}

//func TestOldServer(t *testing.T) {
//	assert := assert.New(t)
//	server := test.New()
//	clusterName := "test_client"
//	config := config.NewCluster()
//	config.Name = clusterName
//	config.StaticEndpoints = "127.0.0.1:10000"
//
//	manager := upstream.NewClusterManager()
//	assert.Nil(manager.InitService(config))
//
//	client := SFactory(
//		Cluster(manager.Cluster(clusterName)),
//		Codec(codec.NewProtoCodec()),
//	)
//
//	go func() {
//		if err := server.Start(10000); err != nil {
//			panic(err)
//		}
//	}()
//
//	time.Sleep(time.Millisecond * 100)
//
//	wg := sync.WaitGroup{}
//	for i := 0; i < 100; i++ {
//		wg.Add(1)
//		i := i
//		go func() {
//			defer wg.Done()
//			m := fmt.Sprintf("%d", i)
//			response := &old.EchoResponse{}
//			request := &old.EchoRequest{
//				Message: proto.String(m),
//			}
//			err := client.Client("echo.EchoService.Echo").Invoke(context.TODO(), request, response)
//			assert.Nil(err)
//			assert.EqualValues(m, response.GetResponse())
//		}()
//	}
//	wg.Wait()
//}
