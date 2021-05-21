package file

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yunfeiyang1916/toolkit/framework/config/source"
)

func TestFile(t *testing.T) {
	data := []byte(`{"foo": "bar"}`)
	path := filepath.Join(os.TempDir(), fmt.Sprintf("file.%d", time.Now().UnixNano()))
	fh, err := os.Create(path)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		fh.Close()
		os.Remove(path)
	}()

	_, err = fh.Write(data)
	if err != nil {
		t.Error(err)
	}

	f := NewSource(WithPath(path))
	c, err := f.Read()
	if err != nil {
		t.Error(err)
	}
	t.Logf("%+v", c)
	if string(c.Data) != string(data) {
		t.Error("data from file does not match")
	}
}

func TestFormat(t *testing.T) {
	opts := source.NewOptions()
	e := opts.Encoder

	testCases := []struct {
		p string
		f string
	}{
		{"/foo/bar.json", "json"},
		{"/foo/bar.yaml", "yaml"},
		{"/foo/bar.xml", "xml"},
		{"/foo/bar.conf.ini", "ini"},
		{"conf", e.String()},
	}

	for _, d := range testCases {
		f := format(d.p, e)
		if f != d.f {
			t.Fatalf("%s: expected %s got %s", d.p, d.f, f)
		}
	}

}
