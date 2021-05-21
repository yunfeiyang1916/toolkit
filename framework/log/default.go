package log

import (
	"os"

	"github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/rolling"
)

func New(path string) *logging.Logger {
	l, _ := logging.NewJSON(path, rolling.HourlyRolling)
	l.SetFlags(0)
	l.SetPrintLevel(false)
	l.SetHighlighting(false)
	l.SetOutputByName(path)
	l.SetTimeFmt(logging.TIMEMICRO)
	return l
}

func Stdout() *logging.Logger {
	s := logging.New()
	s.SetFlags(0)
	s.SetPrintLevel(false)
	s.SetHighlighting(false)
	s.SetOutput(os.Stdout)
	s.SetTimeFmt(logging.TIMEMICRO)
	return s
}

func Noop() *logging.Logger {
	n := logging.New()
	n.SetFlags(0)
	n.SetPrintLevel(false)
	n.SetHighlighting(false)
	n.SetTimeFmt(logging.TIMEMICRO)
	out, _ := os.OpenFile(os.DevNull, os.O_APPEND|os.O_WRONLY, 0600)
	n.SetOutput(out)
	return n
}
