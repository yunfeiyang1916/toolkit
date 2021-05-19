package gid

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert.NotNil(t, New())
}
func TestGId_NewV1(t *testing.T) {
	g := NewGId()
	id := g.NewV1()
	s := fmt.Sprintf("%x", id)
	x := StrToUint64(s)
	assert.Equal(t, x, id)
}

func TestParse(t *testing.T) {
	id := New()
	s := fmt.Sprintf("%x", id)
	want := UnixFromUint64(id)
	got := UnixFromStr(s)
	assert.Equal(t, want, got)
	t.Log(want, got)
}

func TestIpCode(t *testing.T) {
	id := New()
	s := fmt.Sprintf("%x", id)
	code := FnvCodeFromStr(s)
	t.Log(code)
}

// BenchmarkNew-4                  10000000               133 ns/op               0 B/op          0 allocs/op
func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		New()
	}
}

// BenchmarkNewParallel-4          20000000                67.0 ns/op             0 B/op          0 allocs/op
func BenchmarkNewParallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(
		func(pb *testing.PB) {
			for pb.Next() {
				New()
			}
		},
	)
}

func TestRandomFromUint64(t *testing.T) {
	id := New()
	random := RandomFromUint64(id)
	t.Log(random)
	assert.LessOrEqual(t, random, uint32(0xfffff))
}
