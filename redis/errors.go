package redis

import (
	"errors"
)

// Errors code, used as stat upload
const (
	redisSuccess       int = 0   //
	redisError         int = 300 //
	redisConnError     int = 301
	redisConnExhausted int = 302
	redisTimeout       int = 303
)

var (
	ErrConnExhausted = errors.New("redis: connection exhausted, please retry")
	ErrTimeout       = errors.New("redis: i/o timeout, please retry")
)
