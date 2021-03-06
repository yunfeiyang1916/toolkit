// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// socket is an autogenerated mock type for the socket type
type Socket struct {
	mock.Mock
}

// Call provides a mock function with given fields: endpoint, header, body
func (_m *Socket) Call(endpoint string, header map[string]string, body []byte) ([]byte, error) {
	ret := _m.Called(endpoint, header, body)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(string, map[string]string, []byte) []byte); ok {
		r0 = rf(endpoint, header, body)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, map[string]string, []byte) error); ok {
		r1 = rf(endpoint, header, body)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Close provides a mock function with given fields:
func (_m *Socket) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
