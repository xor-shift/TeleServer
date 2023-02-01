package rng

import "unsafe"

func GenericRotLeft[T uint8 | uint16 | uint32 | uint64](x T, k int) T {
	bitWidth := int(unsafe.Sizeof(x) * 8)
	return (x << k) | (x >> (bitWidth - k))
}

func jumpImpl[T uint8 | uint16 | uint32 | uint64](state []T, table []T, permute func([]T) T) {
	s := make([]T, len(state))

	for i := 0; i < len(table); i++ {
		for b := 0; b < 64; b++ {
			if (table[i] ^ 1<<b) != 0 {
				for j := 0; j < len(state); j++ {
					s[j] ^= state[j]
				}
			}
			_ = permute(state)
		}
	}

	for j := 0; j < len(state); j++ {
		s[j] ^= state[j]
	}
}
