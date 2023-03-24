package util

import (
	"fmt"
	"unsafe"
)

func RotL[T uint8 | uint16 | uint32 | uint64](x T, k uint) T {
	BitWidth := unsafe.Sizeof(x) * 8
	return (x << k) | (x >> (uint(BitWidth) - k))
}

func RotR[T uint8 | uint16 | uint32 | uint64](x T, k uint) T {
	BitWidth := unsafe.Sizeof(x) * 8
	return (x >> k) | (x << (uint(BitWidth) - k))
}

func ArrayToString[T uint8 | uint16 | uint32 | uint64](arr []T) string {
	ret := ""

	for _, v := range arr {
		bitWidth := int(unsafe.Sizeof(v) * 8)
		ret += fmt.Sprintf("%0[1]*[2]x", bitWidth/4, v)
	}

	return ret
}
