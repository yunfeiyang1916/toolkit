package ecode

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	var _ error = New(1)
	var _ error = New(2)
	var _ error = New(1)
}

func TestErrMessage(t *testing.T) {
	e1 := New(3)
	if e1.Error() != "3" {
		t.Logf("ecode message should be `3`")
		t.FailNow()
	}
	if e1.Message() != "3" {
		t.Logf("unregistered ecode message should be ecode number")
		t.FailNow()
	}
	Register(map[int]string{3: "testErr"})
	Register(map[int]string{3: "testErr override"})
	if e1.Message() != "testErr override" {
		t.Logf("registered ecode message should be `testErr`")
		t.FailNow()
	}
}

func TestCause(t *testing.T) {
	e1 := New(4)
	var err error = e1
	e2 := Cause(err)
	if e2.Code() != 4 {
		t.Logf("parsed error code should be 4")
		t.FailNow()
	}
}

func TestInt(t *testing.T) {
	e1 := Int(1)
	if e1.Code() != 1 {
		t.Logf("int parsed error code should be 1")
		t.FailNow()
	}
	if e1.Error() != "1" || e1.Message() != "1" {
		t.Logf("int parsed error string should be `1`")
		t.FailNow()
	}
}

func TestString(t *testing.T) {
	eStr := String("123")
	if eStr.Code() != 123 {
		t.Logf("string parsed error code should be 123")
		t.FailNow()
	}
	if eStr.Error() != "123" || eStr.Message() != "123" {
		t.Logf("string parsed error string should be `123`")
		t.FailNow()
	}
	eStr = String("test")
	if eStr.Code() != 500 {
		t.Logf("invalid string parsed error code should be 500")
		t.FailNow()
	}
	if eStr.Error() != "500" || eStr.Message() != "500" {
		t.Logf("invalid string parsed error string should be `500`")
		t.FailNow()
	}
}
func TestRegister(t *testing.T) {
	e := New(100)
	e2 := New(200)
	Register(map[int]string{
		100: "err1",
		200: "err2",
	})
	// register override last value
	Register(map[int]string{
		100: "err11",
	})
	// parallel register
	wg := sync.WaitGroup{}
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Register(map[int]string{
				100: "err11",
			})
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			Register(map[int]string{
				500: "err500",
			})
			ServerErr.Message()
		}()
	}
	wg.Wait()
	assert.Equal(t, "err11", e.Message())
	assert.Equal(t, "err2", e2.Message())
}

func TestError(t *testing.T) {
	e1 := Error(2001, "err1")
	e2 := Error(2002, "err2")
	assert.Equal(t, "err1", e1.Message())
	assert.Equal(t, 2002, e2.Code())
}
