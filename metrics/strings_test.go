package metrics

import (
	"bytes"
	"reflect"
	"testing"
)

func Test_builtinToString(t *testing.T) {
	type args struct {
		k interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := builtinToString(tt.args.k); got != tt.want {
				t.Errorf("builtinToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_floatToString(t *testing.T) {
	type args struct {
		f float64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := floatToString(tt.args.f); got != tt.want {
				t.Errorf("floatToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getBuffer(t *testing.T) {
	tests := []struct {
		name  string
		wantB *bytes.Buffer
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotB := getBuffer(); !reflect.DeepEqual(gotB, tt.wantB) {
				t.Errorf("getBuffer() = %v, want %v", gotB, tt.wantB)
			}
		})
	}
}

func Test_getMetricName(t *testing.T) {
	type args struct {
		name string
		tags []interface{}
	}
	tests := []struct {
		name  string
		args  args
		wantS string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotS := getMetricName(tt.args.name, tt.args.tags); gotS != tt.wantS {
				t.Errorf("getMetricName() = %v, want %v", gotS, tt.wantS)
			}
		})
	}
}

func Test_mapToString(t *testing.T) {
	type args struct {
		a      map[string]string
		except string
	}
	tests := []struct {
		name  string
		args  args
		wantS string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotS := mapToString(tt.args.a, tt.args.except); gotS != tt.wantS {
				t.Errorf("mapToString() = %v, want %v", gotS, tt.wantS)
			}
		})
	}
}
func Benchmark_getMetricName(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			getMetricName("test-timer", []interface{}{"aproject", "1234", "a", 1, "b", 2})
		}
	})
}
