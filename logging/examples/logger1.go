package main

import (
	"github.com/yunfeiyang1916/toolkit/logging"
)

func init() {
	log = logging.NewLogger(&logging.Options{
		Level: "info",
	}, "log1.log", "log2.log")
}

func main() {
	log.Infof("This message will print into log1.log")
	logging.Log("log2").Infof("This message will print into log2.log")
	logging.Log("log1").Debugf("This message will  not print into log1.log, because logger Level is info")
}
