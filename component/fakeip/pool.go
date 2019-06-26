package fakeip

import (
	"errors"
	"math/rand"
	"net"
	"sync"
)

// Pool is a implementation about fake ip generator without storage
type Pool struct {
	max    uint32
	min    uint32
	offset int
	mux    *sync.Mutex
}

// Get return a new fake ip
func (p *Pool) Get() net.IP {
	p.mux.Lock()
	defer p.mux.Unlock()
	offset := rand.Intn(p.offset)
	ip := uintToIP(p.min + uint32(offset))
	return ip
}

func ipToUint(ip net.IP) uint32 {
	v := uint32(ip[0]) << 24
	v += uint32(ip[1]) << 16
	v += uint32(ip[2]) << 8
	v += uint32(ip[3])
	return v
}

func uintToIP(v uint32) net.IP {
	return net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// New return Pool instance
func New(ipnet *net.IPNet) (*Pool, error) {
	min := ipToUint(ipnet.IP) + 1

	ones, bits := ipnet.Mask.Size()
	total := 1<<uint(bits-ones) - 2

	if total <= 0 {
		return nil, errors.New("ipnet don't have valid ip")
	}

	max := min + uint32(total)
	return &Pool{
		min:    min,
		max:    max,
		offset: total,
		mux:    &sync.Mutex{},
	}, nil
}
