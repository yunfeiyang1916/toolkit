package main

import (
	"github.com/yunfeiyang1916/toolkit/logging"
)

func init() {
	log = logging.NewLogger(&logging.Options{
		DisableTimestamp: true,
		DisableLevel:     true,
	})
}

func main() {
	log.Infof("This message will print into stdout")
}
