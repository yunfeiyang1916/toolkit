package main

import (
	"time"

	. "github.com/yunfeiyang1916/toolkit/utils"
)

func doTest() {

	for i := 1; i < 100; i++ {
		st := NewStatEntry("Test")
		st.End("Info", -1)
		time.Sleep(1 * time.Second)
	}
}

func doTest2() {

	for {
		for i := 1; i < 100; i++ {
			st := NewStatEntry("Test")
			st.End("ev1", 0)
			st = NewStatEntry("Test")
			st.End("ev2", 0)
			st = NewStatEntry("Test")
			st.End("ev3", 0)
			st = NewStatEntry("Test")
			st.End("ev3", 3)
			st = NewStatEntry("Test")
			st.End("ev3", 2)
			st = NewStatEntry("Test")
			st.End("ev3", 100)
		}
		time.Sleep(1 * time.Second)
	}
}

func main() {
	AddSuccCode(3)
	go doTest()
	go doTest2()
	time.Sleep(1 * time.Hour)
}
