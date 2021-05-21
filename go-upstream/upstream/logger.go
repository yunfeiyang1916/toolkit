package upstream

import (
	log "github.com/yunfeiyang1916/toolkit/logging"
)

var (
	logging *log.Logger
)

func init() {
	logging = log.New()
}

func SetLogger(l *log.Logger) {
	logging = l
}
