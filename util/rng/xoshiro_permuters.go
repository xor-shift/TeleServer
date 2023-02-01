package rng

// permutes a [2]uint32 state according to xoroshiro64*
// https://prng.di.unimi.it/xoroshiro64star.c
func xoroshiro64SPermuteState(s []uint32) (result uint32) {
	s0 := s[0]
	s1 := s[1]
	result = s0 * 0x9E3779BB

	s1 ^= s0
	s[0] = GenericRotLeft(s0, 26) ^ s1 ^ (s1 << 9)
	s[1] = GenericRotLeft(s1, 13)

	return
}

// permutes a [2]uint32 state according to xoroshiro64**
// https://prng.di.unimi.it/xoroshiro64starstar.c
func xoroshiro64SSPermuteState(s []uint32) (result uint32) {
	s0 := s[0]
	s1 := s[1]
	result = GenericRotLeft(s0*0x9E3779BB, 5) * 5

	s1 ^= s0
	s[0] = GenericRotLeft(s0, 26) ^ s1 ^ (s1 << 9)
	s[1] = GenericRotLeft(s1, 13)

	return
}

// permutes a [4]uint32 state according to xoshiro128+
// https://prng.di.unimi.it/xoshiro128plus.c
func xoshiro128PPermuteState(s []uint32) (result uint32) {
	result = s[0] + s[3]

	t := s[1] << 9

	s[2] ^= s[0]
	s[3] ^= s[1]
	s[1] ^= s[2]
	s[0] ^= s[3]

	s[2] ^= t

	s[3] = GenericRotLeft(s[3], 11)

	return result
}

// permutes a [4]uint32 state according to xoshiro128**
// https://prng.di.unimi.it/xoshiro128starstar.c
func xoshiro128SSPermuteState(s []uint32) (result uint32) {
	result = GenericRotLeft(s[1]*5, 7) * 9

	t := s[1] << 9

	s[2] ^= s[0]
	s[3] ^= s[1]
	s[1] ^= s[2]
	s[0] ^= s[3]

	s[2] ^= t

	s[3] = GenericRotLeft(s[3], 11)

	return result
}

// permutes a [4]uint32 state according to xoshiro128++
// https://prng.di.unimi.it/xoshiro128plusplus.c
func xoshiro128PPPermuteState(s []uint32) (result uint32) {
	result = GenericRotLeft(s[0]+s[3], 7) + s[0]

	t := s[1] << 9

	s[2] ^= s[0]
	s[3] ^= s[1]
	s[1] ^= s[2]
	s[0] ^= s[3]

	s[2] ^= t

	s[3] = GenericRotLeft(s[3], 11)

	return result
}

// permutes a [2]uint64 state according to xoroshiro128+
// https://prng.di.unimi.it/xoroshiro128plus.c
func xoroshiro128PPermuteState(s []uint64) (result uint64) {
	s0 := s[0]
	s1 := s[1]
	result = s0 + s1

	s1 ^= s0
	s[0] = GenericRotLeft(s0, 24) ^ s1 ^ (s1 << 16)
	s[1] = GenericRotLeft(s1, 37)

	return
}

// permutes a [2]uint64 state according to xoroshiro128**
// https://prng.di.unimi.it/xoroshiro128starstar.c
func xoroshiro128SSPermuteState(s []uint64) (result uint64) {
	s0 := s[0]
	s1 := s[1]
	result = GenericRotLeft(s0*5, 7) * 9

	s1 ^= s0
	s[0] = GenericRotLeft(s0, 24) ^ s1 ^ (s1 << 16)
	s[1] = GenericRotLeft(s1, 37)

	return
}

// permutes a [2]uint64 state according to xoroshiro128++
// https://prng.di.unimi.it/xoroshiro128plusplus.c
func xoroshiro128PPPermuteState(s []uint64) (result uint64) {
	s0 := s[0]
	s1 := s[1]
	result = GenericRotLeft(s0+s1, 17) + s0

	s1 ^= s0
	s[0] = GenericRotLeft(s0, 49) ^ s1 ^ (s1 << 21)
	s[1] = GenericRotLeft(s1, 28)

	return
}

// permutes a [4]uint64 state according to xoshiro256+
// https://prng.di.unimi.it/xoshiro256plusplus.c
func xoshiro256PPermuteState(s []uint64) (result uint64) {
	result = s[0] + s[3]

	t := s[1] << 17

	s[2] ^= s[0]
	s[3] ^= s[1]
	s[1] ^= s[2]
	s[0] ^= s[3]

	s[2] ^= t

	s[3] = GenericRotLeft(s[3], 45)

	return
}

// permutes a [4]uint64 state according to xoshiro256**
// https://prng.di.unimi.it/xoshiro256plusplus.c
func xoshiro256SSPermuteState(s []uint64) (result uint64) {
	result = GenericRotLeft(s[1]*5, 7) * 9

	t := s[1] << 17

	s[2] ^= s[0]
	s[3] ^= s[1]
	s[1] ^= s[2]
	s[0] ^= s[3]

	s[2] ^= t

	s[3] = GenericRotLeft(s[3], 45)

	return
}

// permutes a [4]uint64 state according to xoshiro256++
// https://prng.di.unimi.it/xoshiro256plusplus.c
func xoshiro256PPPermuteState(s []uint64) (result uint64) {
	result = GenericRotLeft(s[0]+s[3], 23) + s[0]

	t := s[1] << 17

	s[2] ^= s[0]
	s[3] ^= s[1]
	s[1] ^= s[2]
	s[0] ^= s[3]

	s[2] ^= t

	s[3] = GenericRotLeft(s[3], 45)

	return
}
