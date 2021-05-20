package main

import (
	"github.com/yunfeiyang1916/toolkit/logging"
)

var (
	log *logging.Logger
)

func init() {
	log = logging.NewLogger(&logging.Options{})
}

func main() {
	log.Debugf("This is debug")
}
