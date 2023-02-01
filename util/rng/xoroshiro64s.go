package rng

type Xoroshiro64SState struct {
	State [2]uint32
}

func NewXoroshiro64S() *Xoroshiro64SState {
	return &Xoroshiro64SState{
		State: [2]uint32{0, 0},
	}
}

func (state *Xoroshiro64SState) Next() uint32 {
	return xoroshiro64SPermuteState(state.State[:])
}
