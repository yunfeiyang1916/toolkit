// Package source is the interface for sources
package source

import (
	"crypto/md5" // #nosec
	"fmt"
	"time"
)

type Source interface {
	Read() (*ChangeSet, error)
	Watch() (Watcher, error)
	String() string
}

type Watcher interface {
	Next() (*ChangeSet, error)
	Stop() error
}

type ChangeSet struct {
	Data      []byte
	Checksum  string
	Format    string
	Source    string
	Timestamp time.Time
}

func (c *ChangeSet) Sum() string {
	h := md5.New() // #nosec
	h.Write(c.Data)
	return fmt.Sprintf("%x", h.Sum(nil))
}
