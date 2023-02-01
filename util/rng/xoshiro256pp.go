package rng

import (
	"fmt"
)

type Xoshiro256PPState struct {
	State [4]uint64
}

func NewXoshiro256PP() *Xoshiro256PPState {
	state := Xoshiro256PPState{
		State: [4]uint64{0, 0, 0, 0},
	}

	return &state
}

func (state *Xoshiro256PPState) Next() uint64 {
	return xoshiro256PPPermuteState(state.State[:])
}

func (state *Xoshiro256PPState) Jump128() {
	var jump = [4]uint64{
		0x180ec6d33cfd0aba,
		0xd5a61266f0c9392c,
		0xa9582618e03fc9aa,
		0x39abdc4529b1661c,
	}

	jumpImpl(state.State[:], jump[:], xoshiro256PPPermuteState)
}

func (state *Xoshiro256PPState) Jump192() {
	var jump = [4]uint64{
		0x76e15d3efefdcbbf,
		0xc5004e441c522fb3,
		0x77710069854ee241,
		0x39109bb02acbe635,
	}

	jumpImpl(state.State[:], jump[:], xoshiro256PPPermuteState)
}

func (state *Xoshiro256PPState) String() string {
	s := ""

	for i := 0; i < 4; i++ {
		s += fmt.Sprintf("%016X", state.State[0])
	}

	return s
}
