package utils

import (
	"testing"
	"time"
)

func TestProcessStatInfo(t *testing.T) {
	//SetStat("./stat2.log", "test")
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
		break
	}
}

func BenchmarkStatInfo(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st := NewStatEntry("test")
		st.End("ev", 0)
	}
}

func BenchmarkStatInfoParallel(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			st := NewStatEntry("test")
			st.End("ev", 0)
		}

	})
}
