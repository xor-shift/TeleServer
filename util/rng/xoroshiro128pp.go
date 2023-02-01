package rng

type Xoroshiro128PPState struct {
	State [2]uint64
}

func NewXoroshiro128PP() *Xoroshiro128PPState {
	state := Xoroshiro128PPState{
		State: [2]uint64{0, 0},
	}

	return &state
}

func (state *Xoroshiro128PPState) Next() uint64 {
	return xoroshiro128PPPermuteState(state.State[:])
}

func (state *Xoroshiro128PPState) Jump64() {
	jump := [2]uint64{0x2bd7a6a6e99c2ddc, 0x0992ccaf6a6fca05}

	s0 := uint64(0)
	s1 := uint64(0)
	for i := 0; i < len(jump); i++ {
		for b := 0; b < 64; b++ {
			if (jump[i] & (uint64(1) << b)) != 0 {
				s0 ^= state.State[0]
				s1 ^= state.State[1]
			}
			_ = state.Next()
		}
	}

	state.State[0] = s0
	state.State[1] = s1
}

func (state *Xoroshiro128PPState) Jump96() {
	jump := [2]uint64{0x360fd5f2cf8d5d99, 0x9c6e6877736c46e3}

	s0 := uint64(0)
	s1 := uint64(0)
	for i := 0; i < len(jump); i++ {
		for b := 0; b < 64; b++ {
			if (jump[i] & uint64(1) << b) != 0 {
				s0 ^= state.State[0]
				s1 ^= state.State[1]
			}
			_ = state.Next()
		}
	}

	state.State[0] = s0
	state.State[1] = s1
}
