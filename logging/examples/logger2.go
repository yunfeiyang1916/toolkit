package main

import (
	"github.com/yunfeiyang1916/toolkit/logging"
)

func init() {
	log = logging.NewLogger(&logging.Options{
		TimesFormat: logging.TIMESECOND,
	})
}

func main() {
	log.Infof("This message will print into stdout")
}
