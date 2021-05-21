// Reply Helpers
//
//
// Bool, Int, Bytes, String, Strings and Values 函数把Do的返回值转换成具体的类型。
// 这些函数的第二个参数是由Do函数返回error类型，日和error不是nil，那么这些helper
// 函数就直接返回这个错误，如果error是nil，那么函数把reply转换成特定的类型：
//
//  exists, err := redis.Bool(c.Do("EXISTS", "foo"))
//  if err != nil {
//      // handle error return from c.Do or type conversion error.
//  }
//
//
//  var value1 int
//  var value2 string
//  reply, err := redis.Values(c.Do("MGET", "key1", "key2"))
//  if err != nil {
//      // handle error
//  }
//
package redis

import (
	"github.com/garyburd/redigo/redis"
)

// Bool 是一个helper函数，它把一个命令的reply转换成bool。如果err不是nil，Bool返回false，err。
// 否则Bool把reply按照下列方式转换成bool：
//
//  Reply type      Result
//  integer         value != 0, nil
//  bulk string     strconv.ParseBool(reply)
//  nil             false, ErrNil
//  other           false, error
func Bool(reply interface{}, err error) (bool, error) {
	return redis.Bool(reply, err)
}

// ByteSlices is a helper that converts an array command reply to a [][]byte.
// If err is not equal to nil, then ByteSlices returns nil, err. Nil array
// items are stay nil. ByteSlices returns an error if an array item is not a
// bulk string or nil.
func ByteSlices(reply interface{}, err error) ([][]byte, error) {
	return redis.ByteSlices(reply, err)
}

func Bytes(reply interface{}, err error) ([]byte, error) {
	return redis.Bytes(reply, err)
}

func Float64(reply interface{}, err error) (float64, error) {
	return redis.Float64(reply, err)
}

func Int(reply interface{}, err error) (int, error) {
	return redis.Int(reply, err)
}

func Int64(reply interface{}, err error) (int64, error) {
	return redis.Int64(reply, err)
}

func Int64Map(result interface{}, err error) (map[string]int64, error) {
	return redis.Int64Map(result, err)
}

// IntMap is a helper that converts an array of strings (alternating key, value)
// into a map[string]int. The HGETALL commands return replies in this format.
// Requires an even number of values in result.
func IntMap(result interface{}, err error) (map[string]int, error) {
	return redis.IntMap(result, err)
}

func Ints(reply interface{}, err error) ([]int, error) {
	return redis.Ints(reply, err)
}

// String is a helper that converts a command reply to a string. If err is not
// equal to nil, then String returns "", err. Otherwise String converts the
// reply to a string as follows:
//
//  Reply type      Result
//  bulk string     string(reply), nil
//  simple string   reply, nil
//  nil             "",  ErrNil
//  other           "",  error
func String(reply interface{}, err error) (string, error) {
	return redis.String(reply, err)
}

// StringMap is a helper that converts an array of strings (alternating key, value)
// into a map[string]string. The HGETALL and CONFIG GET commands return replies in this format.
// Requires an even number of values in result.
func StringMap(result interface{}, err error) (map[string]string, error) {
	return redis.StringMap(result, err)
}

// Strings is a helper that converts an array command reply to a []string. If
// err is not equal to nil, then Strings returns nil, err. Nil array items are
// converted to "" in the output slice. Strings returns an error if an array
// item is not a bulk string or nil.
func Strings(reply interface{}, err error) ([]string, error) {
	return redis.Strings(reply, err)
}

func Uint64(reply interface{}, err error) (uint64, error) {
	return redis.Uint64(reply, err)
}

// Values is a helper that converts an array command reply to a []interface{}.
// If err is not equal to nil, then Values returns nil, err. Otherwise, Values
// converts the reply as follows:
//
//  Reply type      Result
//  array           reply, nil
//  nil             nil, ErrNil
//  other           nil, error
func Values(reply interface{}, err error) ([]interface{}, error) {
	return redis.Values(reply, err)
}
