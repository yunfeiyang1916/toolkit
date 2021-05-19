package metrics

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	metrics "github.com/rcrowley/go-metrics"
)

func TestUtils(t *testing.T) {
	go FalconWithTags(
		metrics.DefaultRegistry, // metrics registry
		time.Second*1,           // interval
		map[string]string{
			"project": "project",
			"dc":      "aliyun",
		},
	)
	AddSuccessCode(map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0, 7: 0, 8: 0, 190: 0})
	c := 0
	SetDefaultRergistryTags(map[string]string{"project": "project_aaa", "dc": "dc2"})
	go func() {
		for {
			begin := time.Now().Add(time.Duration(-(rand.Int() % 1000)) * time.Millisecond)
			Gauge("metricname", c, "code", 0, "a", "b", "c", 10)
			c++
			// time.Sleep(1 * time.Millisecond)
			Timer("metrictimer", begin, TagCode, rand.Int()%10, "a", "aa", "b", "bb")
			Timer("metrictimer-code0", begin, TagCode, 0, "a", "aa", "b", "bb")
			Timer("metrictimer-nocode", begin, "aaa", "aaaaaa")
			Timer("metrictimer-notag", begin)
			Meter("meter", 1, "a", "bb")
			Meter("meter-notag", 1)
			Timer("metrictimer-with-comment", begin, TagCode, 0, TagComment, map[int]string{0: "成功", 1: "失败"})
		}
	}()
	for {
		begin := time.Now().Add(time.Duration(-(rand.Int() % 10000)) * time.Millisecond)
		Gauge("metricname", c, "code", 0, "a", "b", "c", 10)
		c++
		// time.Sleep(1 * time.Millisecond)
		Timer("metrictimer", begin, TagCode, rand.Int()%10, "a", "aa", "b", "bb")
		Timer("metrictimer-code0", begin, TagCode, 0, "a", "aa", "b", "bb")
		Timer("metrictimer-nocode", begin, "aaa", "aaaaaa")
		Timer("metrictimer-notag", begin)
		Meter("meter", 1, "a", "bb")
		Meter("meter-notag", 1)
		Timer("metrictimer-with-comment", begin, TagCode, 0, TagComment, map[int]string{0: "成功", 1: "失败"})
	}
}

func TestLongTimeUtils(t *testing.T) {
	go FalconWithTags(
		metrics.DefaultRegistry, // metrics registry
		time.Second*1,           // interval
		map[string]string{
			"project": "project",
			"dc":      "aliyun",
		},
	)
	AddSuccessCode(map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0, 7: 0, 8: 0, 190: 0})
	SetDefaultRergistryTags(map[string]string{"project": "project_aaa", "dc": "dc2"})
	TimerDuration("timer-duration", 10*time.Second, TagCode, 0)
	for {
		TimerDuration("timer-duration", 0*time.Millisecond, TagCode, 1)
		//time.Sleep(1 * time.Second)
	}
}

func TestCombineMaps(t *testing.T) {
	a := map[string]string{"project": "test-project", "dc": "ali"}
	b := map[string]string{"a": "b", "c": "d"}
	fmt.Printf("%s\n", combineMaps(a, b))
}

func BenchmarkBenchMeter(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Meter("test-meter", 100, "project", "1234")
	}
}

func BenchmarkBenchGauge(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Gauge("test-gauge", 100, "project", "1234")
	}
}

func BenchmarkBenchTimer(b *testing.B) {
	go FalconWithTags(
		metrics.DefaultRegistry, // metrics registry
		time.Second*60,          // interval
		map[string]string{
			"project": "project",
			"dc":      "aliyun",
		},
	)
	beginTime := time.Now()
	time.Sleep(1 * time.Second)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Timer("test-timer", beginTime, "aproject", "1234")
		}
	})
}
