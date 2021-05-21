package ikiosocket

import (
	"testing"
	"time"
)

func TestTimer(t *testing.T) {
	timer := NewTimer(time.Second, func() {
		t.Log("End")
	})
	timer.Start()
	timer.Stop()
	time.Sleep(time.Second * 3)
}
