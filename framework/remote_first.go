package framework

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yunfeiyang1916/toolkit/framework/config"
	"github.com/yunfeiyang1916/toolkit/framework/utils"
	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
	"github.com/yunfeiyang1916/toolkit/logging"
)

type statusCallback func()

var (
	initDone int32
)

var rfc = &remoteFirstControl{}

type remoteFirstControl struct {
	curStatus bool
	sync.Mutex
	enableHandlers  []statusCallback
	disableHandlers []statusCallback
	changedHandlers []statusCallback
}

func initRemoteFirst(sdname string) {
	if !atomic.CompareAndSwapInt32(&initDone, 0, 1) {
		return
	}
	remoteFirstPath := filepath.Join(getRegistryKVPath(sdname), "remote_first")
	val, _, _ := registry.Default.ReadManual(remoteFirstPath)
	curTime := time.Now().Format(utils.TimeFormat)
	fmt.Printf("%s service %s remote_first status %s\n", curTime, sdname, val)
	logging.GenLogf("", val)
	status := strings.ToLower(strings.TrimSpace(val)) == "true"
	rfc.curStatus = status
	w := config.WatchKV(remoteFirstPath)
	go func() {
		for {
			v := w.Next()
			if len(v) == 0 {
				continue
			}
			value := v[remoteFirstPath]
			newStatus := strings.ToLower(strings.TrimSpace(value)) == "true"
			rfc.Lock()
			if rfc.curStatus == newStatus {
				rfc.Unlock()
				continue
			}
			rfc.curStatus = newStatus
			changedCallbacks := make([]statusCallback, len(rfc.changedHandlers))
			copy(changedCallbacks, rfc.changedHandlers)
			enableCallbacks := make([]statusCallback, len(rfc.enableHandlers))
			copy(enableCallbacks, rfc.enableHandlers)
			disableCallbacks := make([]statusCallback, len(rfc.disableHandlers))
			copy(disableCallbacks, rfc.disableHandlers)
			rfc.Unlock()
			for _, f := range changedCallbacks {
				f()
			}
			if newStatus {
				for _, f := range enableCallbacks {
					f()
				}
			} else {
				for _, f := range disableCallbacks {
					f()
				}
			}
		}
	}()
	return
}

func (rfc *remoteFirstControl) status() bool {
	rfc.Lock()
	defer rfc.Unlock()
	return rfc.curStatus
}

//nolint:unused
func (rfc *remoteFirstControl) registerOnEnable(f ...statusCallback) {
	rfc.Lock()
	defer rfc.Unlock()
	if len(rfc.enableHandlers) == 0 {
		rfc.enableHandlers = make([]statusCallback, 0)
	}
	rfc.enableHandlers = append(rfc.enableHandlers, f...)
}

//nolint:unused
func (rfc *remoteFirstControl) registerOnDisable(f ...statusCallback) {
	rfc.Lock()
	defer rfc.Unlock()
	if len(rfc.disableHandlers) == 0 {
		rfc.disableHandlers = make([]statusCallback, 0)
	}
	rfc.disableHandlers = append(rfc.disableHandlers, f...)
}

//nolint:unused
func (rfc *remoteFirstControl) registerOnChanged(f ...statusCallback) {
	rfc.Lock()
	defer rfc.Unlock()
	if len(rfc.changedHandlers) == 0 {
		rfc.changedHandlers = make([]statusCallback, 0)
	}
	rfc.changedHandlers = append(rfc.changedHandlers, f...)
}
