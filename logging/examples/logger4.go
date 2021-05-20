package main

import (
	"github.com/yunfeiyang1916/toolkit/logging"
)

func init() {
	log = logging.NewLogger(&logging.Options{
		Rolling: logging.DAILY,
	}, "log1.log")
}

func main() {
	for {
		log.Infof("This message will print into log1.log")
		l := log.With("key", "test key", "request", "test request")
		l.Infof("message to log")
	}
}
