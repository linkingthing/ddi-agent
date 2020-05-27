package util

import (
	"encoding/binary"
	"math/big"
	"net"
)

func Ipv4ToUint32(ipv4 net.IP) uint32 {
	return binary.BigEndian.Uint32(ipv4.To4())
}

func Ipv6ToBigInt(ipv6 net.IP) *big.Int {
	ipv6Int := big.NewInt(0)
	ipv6Int.SetBytes(ipv6.To16())
	return ipv6Int
}
