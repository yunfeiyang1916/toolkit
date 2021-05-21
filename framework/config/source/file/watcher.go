package file

import (
	"errors"
	"time"

	"github.com/yunfeiyang1916/toolkit/framework/config/source"
	"github.com/yunfeiyang1916/toolkit/logging"
)

type watcher struct {
	f    *file
	exit chan bool
	ch   chan *source.ChangeSet
}

func newWatcher(f *file) (source.Watcher, error) {
	w := &watcher{
		f:    f,
		exit: make(chan bool),
		ch:   make(chan *source.ChangeSet),
	}
	// 10s 读一次
	tick := time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-w.exit:
				logging.GenLogf("file watcher closed, %+v", *f)
				return
			case <-tick.C:
				c, _ := w.f.Read()
				w.ch <- c
			}
		}
	}()
	return w, nil
}

func (w *watcher) Next() (*source.ChangeSet, error) {
	select {
	case <-w.exit:
		return nil, errors.New("watcher stopped")
	case cs := <-w.ch:
		return cs, nil
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
