package rolling

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type RollingFile struct {
	mu sync.Mutex

	closed    bool
	exit      chan struct{}
	syncFlush chan struct{}

	file       *os.File
	current    *bytes.Buffer
	fullBuffer chan *bytes.Buffer

	basePath string
	filePath string
	fileFrag string

	rollMutex sync.RWMutex
	rolling   RollingFormat
}

var ErrClosedRollingFile = errors.New("rolling file is closed")
var ErrBuffer = errors.New("buffer exceeds the limit")

type RollingFormat string

const (
	MonthlyRolling  RollingFormat = "200601"
	DailyRolling                  = "20060102"
	HourlyRolling                 = "2006010215"
	MinutelyRolling               = "200601021504"
	SecondlyRolling               = "20060102150405"
)

const (
	logPageCacheByteSize = 4096
	logPageNumber        = 2
)

func (r *RollingFile) SetRolling(fmt RollingFormat) {
	r.rollMutex.Lock()
	r.rolling = fmt
	r.rollMutex.Unlock()
}

func (r *RollingFile) roll() error {
	r.rollMutex.RLock()
	roll := r.rolling
	r.rollMutex.RUnlock()
	suffix := time.Now().Format(string(roll))
	if r.file != nil {
		if suffix == r.fileFrag {
			return nil
		}
		r.file.Close()
		r.file = nil
	}
	r.fileFrag = suffix
	dir, filename := filepath.Split(r.basePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return err
		}
	}
	if r.fileFrag == "" {
		r.filePath = filepath.Join(dir, filename+".log")
	} else {
		r.filePath = filepath.Join(dir, filename+"-"+r.fileFrag+".log")
	}
	f, err := os.OpenFile(r.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	r.file = f
	return nil
}

func (r *RollingFile) createSymLink(real, sym string) {
	if _, err := os.Lstat(sym); err == nil {
		os.Remove(sym)
	}
	os.Symlink(real, sym)
}

func (r *RollingFile) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.mu.Unlock()
	close(r.exit)
	return nil
}

func (r *RollingFile) Write(b []byte) (n int, err error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return 0, ErrClosedRollingFile
	}

	if r.current == nil {
		r.current = getBuffer()
		if r.current == nil {
			r.mu.Unlock()
			return 0, ErrBuffer
		}
	}
	n, err = r.current.Write(b)
	if r.current.Len() > logPageCacheByteSize {
		buf := r.current
		r.current = nil
		r.mu.Unlock()
		r.fullBuffer <- buf
	} else {
		r.mu.Unlock()
	}
	return
}

func (r *RollingFile) Sync() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return ErrClosedRollingFile
	}
	r.mu.Unlock()

	r.syncFlush <- struct{}{}
	<-r.syncFlush
	return nil
}

func (r *RollingFile) writeBuffer(buff *bytes.Buffer) {
	if buff != nil && buff.Len() > 0 {
		if err := r.roll(); err != nil {
		} else {
			buff.WriteTo(r.file)
		}
	}
}

func (r *RollingFile) flushRoutine() {
	flush := func() {
		readyLen := len(r.fullBuffer)
		for i := 0; i < readyLen; i++ {
			buff := <-r.fullBuffer
			r.writeBuffer(buff)
			putBuffer(buff)
		}
		if r.current != nil {
			r.writeBuffer(r.current)
			putBuffer(r.current)
		}
		r.current = nil
		if r.file != nil {
			r.file.Sync()
		}
	}

	//FIXME better solution ?
	defer func() {
		flush()
		if f := r.file; f != nil {
			r.file = nil
			f.Close()
		}
	}()
	for {
		select {
		case <-r.syncFlush:
			r.mu.Lock()
			flush()
			r.mu.Unlock()
			r.syncFlush <- struct{}{}
		case buff := <-r.fullBuffer:
			r.writeBuffer(buff)
			putBuffer(buff)
		case <-time.After(1 * time.Second):
			r.mu.Lock()
			if len(r.fullBuffer) != 0 {
				r.mu.Unlock()
				continue
			}
			// 清空buffer
			buff := r.current
			if buff == nil {
				r.mu.Unlock()
				continue
			}
			r.current = nil
			r.mu.Unlock()

			r.writeBuffer(buff)
			putBuffer(buff)
		case <-r.exit:
			return

		}
	}
}

func NewRollingFile(basePath string, rolling RollingFormat) (*RollingFile, error) {
	basePath = strings.TrimSuffix(basePath, ".log")
	if _, file := filepath.Split(basePath); file == "" {
		return nil, fmt.Errorf("invalid base-path = %s, file name is required", basePath)
	}
	r := &RollingFile{
		basePath:   basePath,
		rolling:    rolling,
		exit:       make(chan struct{}),
		syncFlush:  make(chan struct{}),
		closed:     false,
		fullBuffer: make(chan *bytes.Buffer, logPageNumber+1),
		current:    getBuffer(),
	}
	// fill ready buffer
	go r.flushRoutine()
	return r, nil
}
