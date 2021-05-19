package gid

import (
	"errors"
	"hash/fnv"
	"math/rand"
	"net"
	"os"
	"sync"
)

// 允许并发生成随机数
type lockedSource struct {
	mut sync.Mutex
	src rand.Source
}

func NewRand(seed int64) *rand.Rand {
	return rand.New(&lockedSource{src: rand.NewSource(seed)})
}

func (r *lockedSource) Int63() (n int64) {
	r.mut.Lock()
	n = r.src.Int63()
	r.mut.Unlock()
	return
}

func (r *lockedSource) Seed(seed int64) {
	r.mut.Lock()
	r.src.Seed(seed)
	r.mut.Unlock()
}

func scoreAddr(iface net.Interface, addr net.Addr) (int, net.IP) {
	var ip net.IP
	if netAddr, ok := addr.(*net.IPNet); ok {
		ip = netAddr.IP
	} else if netIP, ok := addr.(*net.IPAddr); ok {
		ip = netIP.IP
	} else {
		return -1, nil
	}

	var score int
	if ip.To4() != nil {
		score += 300
	}
	if iface.Flags&net.FlagLoopback == 0 && !ip.IsLoopback() {
		score += 100
		if iface.Flags&net.FlagUp != 0 {
			score += 100
		}
	}
	return score, ip
}

// HostIP tries to find an IP that can be used by other machines to reach this machine.
func HostIP() (net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	bestScore := -1
	var bestIP net.IP
	// Select the highest scoring IP as the best IP.
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			// Skip this interface if there is an error.
			continue
		}

		for _, addr := range addrs {
			score, ip := scoreAddr(iface, addr)
			if score > bestScore {
				bestScore = score
				bestIP = ip
			}
		}
	}

	if bestScore == -1 {
		return nil, errors.New("no addresses to listen on")
	}

	return bestIP, nil
}

func HashCode(s []byte) uint32 {
	h := fnv.New32a()
	n, err := h.Write(s)
	if err != nil || n == 0 {
		return 0
	}
	return h.Sum32()
}
func Fnv32a(s []byte) uint32 {
	var offset32 uint32 = 2166136261
	var prime32 uint32 = 16777619
	hash := offset32
	for _, c := range s {
		hash ^= uint32(c)
		hash *= prime32
	}
	return hash
}
func Fnv32aExt(s []byte) uint64 {
	var offset32 uint32 = 2166136261
	var prime32 uint32 = 16777619
	hash := offset32
	for _, c := range s {
		hash ^= uint32(c)
		hash *= prime32
	}
	return uint64(hash)
}

func IpHashCode() uint32 {
	ip, err := HostIP()
	localIp := []byte(ip.String())
	if err != nil || ip.String() == "127.0.0.1" {
		randBuf := make([]byte, 4)
		rand.Read(randBuf)
		localIp = randBuf
	}
	return HashCode(localIp)
}
func HostnameHashCode() uint32 {
	hostname, err := os.Hostname()
	buf := []byte(hostname)
	if err != nil || hostname == "localhost" {
		randBuf := make([]byte, 4)
		rand.Read(randBuf)
		buf = randBuf
	}
	return HashCode(buf)
}

func SplitId(s string) (low string, high string) {
	if len(s) < 16 {
		low = s
		return
	}
	low = s[len(s)-16:]
	high = s[0 : len(s)-16]
	return
}
