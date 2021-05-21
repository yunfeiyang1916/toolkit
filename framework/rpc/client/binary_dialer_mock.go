package client

import (
	"github.com/stretchr/testify/mock"
)

type mockedDialer struct {
	mock.Mock
}

func (_m *mockedDialer) Dial(host string) (socket, error) {
	ret := _m.Called(host)

	var r0 socket
	if rf, ok := ret.Get(0).(func(string) socket); ok {
		r0 = rf(host)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(socket)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(host)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
