package hash

import "zkdilithium-signer/pkg/field"

// PosRCs contains the Poseidon round constants.
var PosRCs [field.PosT * field.PosRF]uint32

// PosInv contains precomputed inverses for MDS: 1/(i+j-1) for i,j in [1, 2*PosT-1]
var PosInv [2*field.PosT - 1]uint32

func init() {
	// Generate round constants using Grain LFSR
	g := NewGrain()
	for i := 0; i < field.PosT*field.PosRF; i++ {
		PosRCs[i] = g.ReadFe()
	}

	// Generate MDS inverses
	for i := 0; i < 2*field.PosT-1; i++ {
		PosInv[i] = field.Inv(uint32(i + 1))
	}
}

// poseidonRound applies one round of the Poseidon permutation.
func poseidonRound(state []uint32, r int) {
	// Add round constants
	for i := 0; i < field.PosT; i++ {
		state[i] = field.Add(state[i], PosRCs[field.PosT*r+i])
	}

	// S-box: x -> x^(-1) (modular inverse)
	for i := 0; i < field.PosT; i++ {
		state[i] = field.Inv(state[i])
	}

	// MDS matrix multiplication: M_ij = 1/(i+j-1)
	old := make([]uint32, field.PosT)
	copy(old, state)
	for i := 0; i < field.PosT; i++ {
		var acc uint64
		for j := 0; j < field.PosT; j++ {
			acc += uint64(PosInv[i+j]) * uint64(old[j])
		}
		state[i] = uint32(acc % field.Q)
	}
}

// PoseidonPerm applies the full Poseidon permutation to state in place.
func PoseidonPerm(state []uint32) {
	for r := 0; r < field.PosRF; r++ {
		poseidonRound(state, r)
	}
}

// Poseidon is a sponge construction using the Poseidon permutation.
type Poseidon struct {
	s         [field.PosT]uint32
	absorbing bool
	i         int
}

// NewPoseidon creates a new Poseidon sponge, optionally with initial values.
func NewPoseidon(initial []uint32) *Poseidon {
	p := &Poseidon{
		absorbing: true,
	}
	if initial != nil {
		p.Write(initial)
	}
	return p
}

// Write absorbs field elements into the sponge.
func (p *Poseidon) Write(fes []uint32) {
	if !p.absorbing {
		panic("cannot write after reading")
	}
	for _, fe := range fes {
		p.s[p.i] = field.Add(p.s[p.i], fe)
		p.i++
		if p.i == field.PosRate {
			PoseidonPerm(p.s[:])
			p.i = 0
		}
	}
}

// Permute applies the permutation if there's pending input.
func (p *Poseidon) Permute() {
	if !p.absorbing {
		panic("cannot permute after reading")
	}
	if p.i != 0 {
		PoseidonPerm(p.s[:])
		p.i = 0
	}
}

// Read squeezes n field elements from the sponge.
func (p *Poseidon) Read(n int) []uint32 {
	if p.absorbing {
		p.absorbing = false
		if p.i != 0 {
			PoseidonPerm(p.s[:])
			p.i = 0
		}
	}

	ret := make([]uint32, 0, n)
	for n > 0 {
		toRead := n
		if toRead > field.PosRate-p.i {
			toRead = field.PosRate - p.i
		}
		ret = append(ret, p.s[p.i:p.i+toRead]...)
		n -= toRead
		p.i += toRead
		if p.i == field.PosRate {
			p.i = 0
			PoseidonPerm(p.s[:])
		}
	}
	return ret
}

// ReadNoMod reads n elements from state without modifying position.
// Used in sampleInBall.
func (p *Poseidon) ReadNoMod(n int) []uint32 {
	if n > field.PosRate {
		panic("ReadNoMod: n > PosRate")
	}
	return p.s[:n]
}

// State returns a pointer to the internal state (for sampleInBall).
func (p *Poseidon) State() *[field.PosT]uint32 {
	return &p.s
}

// ApplyPerm applies the Poseidon permutation to the internal state.
func (p *Poseidon) ApplyPerm() {
	PoseidonPerm(p.s[:])
}
