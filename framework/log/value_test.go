package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBindValuer(t *testing.T) {
	valuer := func() interface{} {
		return ""
	}
	var vals []interface{}
	for i := 0; i < 100; i++ {
		vals = append(vals, Valuer(valuer))
	}

	bindValues(vals)

	for i := 0; i < len(vals); i++ {
		if _, ok := vals[i].(string); !ok && i%2 != 0 {
			assert.Fail(t, "not expect")
		}

		if _, ok := vals[i].(string); ok && i%2 == 0 {
			assert.Fail(t, "not expect")
		}
	}
}

func TestContainsValuer(t *testing.T) {
	valuer := func() interface{} {
		return ""
	}
	var vals []interface{}

	for i := 0; i < 100; i++ {
		vals = append(vals, "")
	}

	assert.False(t, containsValuer(vals))

	for i := 0; i < 100; i++ {
		vals = append(vals, Valuer(valuer))
	}

	assert.True(t, containsValuer(vals))
}

func TestTimestamp(t *testing.T) {
	assert := assert.New(t)
	n := time.Now()
	valuer := Timestamp(func() time.Time {
		return n
	})
	v, ok := valuer().(time.Time)
	assert.True(ok)
	assert.Equal(v, n)
}

func TestTimestampFormat(t *testing.T) {
	DefaultTimestamp()
}
