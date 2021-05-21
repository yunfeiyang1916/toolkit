package consul

import (
	"bytes"
	"errors"
	"strings"
	"time"

	"github.com/yunfeiyang1916/toolkit/logging"

	"github.com/yunfeiyang1916/toolkit/framework/config/encoder"
	"github.com/yunfeiyang1916/toolkit/framework/config/source"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
)

type watcher struct {
	e           encoder.Encoder
	name        string
	stripPrefix string
	ch          chan *source.ChangeSet
	exit        chan bool
}

func newWatcher(key, name, stripPrefix string, e encoder.Encoder) source.Watcher {
	w := &watcher{
		e:           e,
		name:        name,
		stripPrefix: stripPrefix,
		ch:          make(chan *source.ChangeSet),
		exit:        make(chan bool),
	}

	kvMapChan := make(chan map[string]string)
	kvStrChan := make(chan string)
	if len(stripPrefix) > 0 {
		// return map[string]string: key(is path) = value
		kvMapChan = registry.Default.WatchPrefixManual(key)
	} else {
		// return string value for key
		kvStrChan = registry.Default.WatchManual(key)
	}

	go func() {
		for {
			buf := bytes.NewBuffer(nil)
			select {
			case <-w.exit:
				logging.GenLogf("consul watcher closed, key %s", key)
				return
			case d := <-kvMapChan:
				if len(d) == 0 {
					continue
				}
				value := map[string]string{}
				for k, v := range d {
					kk := strings.TrimPrefix(k, strings.TrimPrefix(key, "/"))
					if len(v) == 0 {
						continue
					}
					value[kk] = v
				}
				b, _ := w.e.Encode(value)
				buf.Write(b)
			case d := <-kvStrChan:
				if len(d) == 0 {
					continue
				}
				buf.WriteString(d)
			}
			cs := &source.ChangeSet{
				Timestamp: time.Now(),
				Format:    w.e.String(),
				Source:    w.name,
				Data:      buf.Bytes(),
			}
			cs.Checksum = cs.Sum()
			w.ch <- cs
		}
	}()
	return w
}

func (w *watcher) Next() (*source.ChangeSet, error) {
	select {
	case cs := <-w.ch:
		return cs, nil
	case <-w.exit:
		return nil, errors.New("watcher stopped")
	}
}

func (w *watcher) Stop() error {
	select {
	case <-w.exit:
		return nil
	default:
		close(w.exit)
	}
	return nil
}
