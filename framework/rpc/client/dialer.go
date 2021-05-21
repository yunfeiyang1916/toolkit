package client

import (
	"net"

	"github.com/yunfeiyang1916/toolkit/logging"
)

type dialer interface {
	Dial(host string) (socket, error)
}

type defaultDialer struct {
	opts Options
}

func (d defaultDialer) Dial(host string) (socket, error) {
	conn, err := net.DialTimeout("tcp4", host, d.opts.DialTimeout)
	if err != nil {
		return nil, err
	}
	g := logging.Log(logging.GenLoggerName)
	socket := newIKSocket(g, conn, d.opts.CallOptions.RequestTimeout)
	return socket, nil
}
