package util

import (
	"encoding/binary"
	"math/big"
	"net"
)

func Ipv4StringToUint32(ipv4 string) (uint32, bool) {
	return Ipv4ToUint32(net.ParseIP(ipv4))
}

func Ipv4ToUint32(ipv4 net.IP) (uint32, bool) {
	if ipv4_ := ipv4.To4(); ipv4_ == nil {
		return 0, false
	} else {
		return binary.BigEndian.Uint32(ipv4_), true
	}
}

func Ipv6StringToBigInt(ipv6 string) (*big.Int, bool) {
	return Ipv6ToBigInt(net.ParseIP(ipv6))
}

func Ipv6ToBigInt(ipv6 net.IP) (*big.Int, bool) {
	if ipv6.To4() != nil {
		return nil, false
	}

	ipv6Int := big.NewInt(0)
	ipv6Int.SetBytes(ipv6.To16())
	return ipv6Int, true
}

func OneIpLessThanAnother(one, another string) bool {
	oneIP := net.ParseIP(one)
	anotherIP := net.ParseIP(one)
	if oneIP.To4() != nil && anotherIP.To4() == nil {
		return true
	}

	if oneIP.To4() == nil && anotherIP.To4() != nil {
		return false
	}

	if oneIP.To4() != nil {
		oneUint32, _ := Ipv4ToUint32(oneIP)
		anotherUint32, _ := Ipv4ToUint32(anotherIP)
		return oneUint32 < anotherUint32
	} else {
		oneBigInt, _ := Ipv6ToBigInt(oneIP)
		anotherBigInt, _ := Ipv6ToBigInt(anotherIP)
		return oneBigInt.Cmp(anotherBigInt) == -1
	}
}
