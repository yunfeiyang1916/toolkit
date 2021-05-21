// Package file is a file source. Expected format is json
package file

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/yunfeiyang1916/toolkit/logging"

	"github.com/yunfeiyang1916/toolkit/framework/config/encoder"
	"github.com/yunfeiyang1916/toolkit/framework/config/source"
)

type file struct {
	path string
	opts source.Options
}

func (f *file) Read() (*source.ChangeSet, error) {
	fh, err := os.Open(f.path)
	if err != nil {
		logging.GenLogf("file Read, err:%v", err)
		return nil, err
	}
	defer fh.Close()
	b, err := ioutil.ReadAll(fh)
	if err != nil {
		logging.GenLogf("file Read, err:%v", err)
		return nil, err
	}
	info, err := fh.Stat()
	if err != nil {
		logging.GenLogf("file Read, err:%v", err)
		return nil, err
	}

	cs := &source.ChangeSet{
		Format:    format(f.path, f.opts.Encoder),
		Source:    f.String(),
		Timestamp: info.ModTime(),
		Data:      b,
	}
	cs.Checksum = cs.Sum()
	return cs, nil
}

func (f *file) String() string {
	return "file"
}

func (f *file) Watch() (source.Watcher, error) {
	if _, err := os.Stat(f.path); err != nil {
		return nil, err
	}
	return newWatcher(f)
}

func NewSource(opts ...source.Option) source.Source {
	options := source.NewOptions(opts...)
	var path string
	f, ok := options.Context.Value(filePathKey{}).(string)
	if ok {
		path = f
	}
	return &file{opts: options, path: path}
}

func format(p string, e encoder.Encoder) string {
	parts := strings.Split(p, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return e.String()
}
