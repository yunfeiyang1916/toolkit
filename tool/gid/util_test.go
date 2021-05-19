package gid

import (
	"fmt"
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
)

func TestHostIP(t *testing.T) {
	ip, err := HostIP()
	t.Log(ip, err)
	assert.Nil(t, err)
}

func TestIpHashCode(t *testing.T) {
	code := IpHashCode() % (1 << 16)
	assert.NotNil(t, code)
}
func TestHashCode(t *testing.T) {
	buf := []byte(`hello-world:formatString`)
	v1 := Fnv32a(buf)
	v2 := HashCode(buf)
	assert.Equal(t, v1, v2)
}

func TestHostnameHashCode(t *testing.T) {
	code := HostnameHashCode()
	t.Log(code)
	assert.NotNil(t, code)
}
func TestNginxTid(t *testing.T) {
	s := "68c757d501aaea64faf6058dc"
	low, high := SplitId(s)
	t.Log(low, high)
	t.Log(UnixFromStr(low))
	t.Log(FnvCodeFromStr(low))
	t.Log(RandomFromStr(low))
	h := "localhost"
	code := Fnv32a([]byte(h)) % (1 << 16)
	t.Log(code)
	ss := "sv:op"
	t.Log(Fnv32a([]byte(ss)))
	// assert.Equal(t, high, fmt.Sprintf("%x", Fnv32a([]byte(ss))), "serviceName:op")
	fmt.Println(fmt.Sprintf("%x", New()))
}
func ExampleUnixFromStr() {
	s := "a376611a0fc5a696bbfefc84"
	low, high := SplitId(s)
	fmt.Println(low, high)
	fmt.Println(UnixFromStr(low))
	fmt.Println(time.Unix(UnixFromStr(low), 0))
	fmt.Println(FnvCodeFromStr(low))
	fmt.Println(RandomFromStr(low))

}
