package gid

import (
	"crypto/rand"
	"encoding/hex"
)

// UUID returns a rand UUID string
func UUID() string {
	buf := make([]byte, 16)
	if _, err := rand.Reader.Read(buf[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}

/*

UUID参考

Version 1,基于 timestamp 和 MAC address (RFC 4122)
Version 2,基于 timestamp, MAC address 和 POSIX UID/GID (DCE 1.1)
Version 3, 基于 MD5 hashing (RFC 4122)
Version 4, 基于 random numbers (RFC 4122)
Version 5, 基于 SHA-1 hashing (RFC 4122)

安装 satori/go.uuid 库

# go get github.com/satori/go.uuid
简单使用方法

package main

import (
"fmt"
"github.com/satori/go.uuid"
)

func main() {
	// 创建UUID
	u1 := uuid.Must(uuid.NewV4()).String() //上文介绍的Version 4
	fmt.Printf("UUIDv4: %s\n", u1)
}
*/
