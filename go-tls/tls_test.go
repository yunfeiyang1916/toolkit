package tls

import (
	"sync"
	"testing"

	context "golang.org/x/net/context"
)

func BenchmarkGoID(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			GoID()
		}
	})
}

func BenchmarkRuntimeGoIDSlow(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			GoIDSlow()
		}
	})
}

func TestWith(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go With(func() {
		Set("hello", "world")
		go With(func() {
			if v, ok := Get("hello"); !ok || v != "world" {
				t.Fail()
			}
			wg.Done()
		})()
	})()
	wg.Wait()
}

func TestSet(t *testing.T) {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	Set("hello", "world")
	go func() {
		defer wg.Done()
		Set("hello", "world2")
		if v, ok := Get("hello"); !ok || v != "world2" {
			t.Fatalf("goroutine 2 get unexpected, go %v, expected world2", v)
		}
		Flush()
		if _, ok := Get("hello"); ok {
			t.Fail()
		}

	}()
	wg.Wait()
	if v, ok := Get("hello"); !ok || v != "world" {
		t.Fail()
	}
	Flush()
	if _, ok := Get("hello"); ok {
		t.Fail()
	}
}

func TestWrap(t *testing.T) {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	ctx := context.WithValue(context.Background(), "a", 1)
	SetContext(ctx)
	go Wrap(func() {
		defer wg.Done()
		ctxNew, exist := GetContext()
		if exist != true || ctxNew != ctx {
			t.Fatal("Wrap context error")
		}
		if ctxNew.Value("a") != 1 {
			t.Fatal("Wrap context get value error")
		}
	})()

	wg.Wait()
	Flush()
}

func BenchmarkSet(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Set("hello", "world")
			Get("hello")
			Flush()
		}
	})
}

func BenchmarkWrap(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx := context.WithValue(context.Background(), "a", 1)
			SetContext(ctx)
			Wrap(func() {
				ctxNew, exist := GetContext()
				if exist != true || ctxNew != ctx {
					b.Fatalf("Wrap context error %v", ctxNew)
				}
				if ctxNew.Value("a") != 1 {
					b.Fatal("Wrap context get value error")
				}
			})()
			Flush()
		}
	})
}
