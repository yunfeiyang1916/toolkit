package rpcerror

const (
	Success     int = 0
	Internal    int = 1
	Timeout     int = 2
	BreakerOpen int = 3
	Ratelimit   int = 4
	FromUser    int = 1000
	UnKnown     int = 1001
	Request     int = 1002
)
