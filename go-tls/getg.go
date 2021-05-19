package tls

import (
	"unsafe"
)

func getg() unsafe.Pointer

// G returns current g (the goroutine struct) to user space.
func G() unsafe.Pointer {
	return getg()
}
