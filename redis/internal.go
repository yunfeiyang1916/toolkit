package redis

import (
	"github.com/garyburd/redigo/redis"
)

func redisBool(reply interface{}, err error) (interface{}, error) {
	return redis.Bool(reply, err)
}

func redisByteSlices(reply interface{}, err error) (interface{}, error) {
	return redis.ByteSlices(reply, err)
}

func redisBytes(reply interface{}, err error) (interface{}, error) {
	return redis.Bytes(reply, err)
}

func redisFloat64(reply interface{}, err error) (interface{}, error) {
	return redis.Float64(reply, err)
}

func redisInt(reply interface{}, err error) (interface{}, error) {
	return redis.Int(reply, err)
}

func redisInt64(reply interface{}, err error) (interface{}, error) {
	return redis.Int64(reply, err)
}

func redisInt64Map(reply interface{}, err error) (interface{}, error) {
	return redis.Int64Map(reply, err)
}

func redisIntMap(reply interface{}, err error) (interface{}, error) {
	return redis.IntMap(reply, err)
}

func redisInts(reply interface{}, err error) (interface{}, error) {
	return redis.Ints(reply, err)
}

func redisString(reply interface{}, err error) (interface{}, error) {
	return redis.String(reply, err)
}

func redisStringMap(reply interface{}, err error) (interface{}, error) {
	return redis.StringMap(reply, err)
}

func redisStrings(reply interface{}, err error) (interface{}, error) {
	return redis.Strings(reply, err)
}

func redisUint64(reply interface{}, err error) (uint64, error) {
	return redis.Uint64(reply, err)
}
