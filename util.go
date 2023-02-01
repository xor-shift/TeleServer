package main

import (
	"fmt"
	"unsafe"
)

func ArrayToString[T uint8 | uint16 | uint32 | uint64](arr []T) string {
	ret := ""

	for _, v := range arr {
		bitWidth := int(unsafe.Sizeof(v) * 8)
		ret += fmt.Sprintf("%0[1]*[2]x", bitWidth/4, v)
	}

	return ret
}
