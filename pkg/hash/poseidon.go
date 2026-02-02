package hash

import "zkdilithium-signer/pkg/field"

// PosRCsMont contains the Poseidon round constants in Montgomery form.
var PosRCsMont [field.PosT * field.PosRF]uint32

// PosInvMont contains precomputed inverses for MDS in Montgomery form.
// PosInvMont[i] = (1/(i+1))_M for i in [0, 2*PosT-2]
var PosInvMont [2*field.PosT - 1]uint32

func init() {
	// Generate round constants using Grain LFSR, convert to Montgomery form
	g := NewGrain()
	for i := 0; i < field.PosT*field.PosRF; i++ {
		PosRCsMont[i] = field.ToMont(g.ReadFe())
	}

	// Generate MDS inverses in Montgomery form
	for i := 0; i < 2*field.PosT-1; i++ {
		PosInvMont[i] = field.ToMont(field.Inv(uint32(i + 1)))
	}
}

// poseidonRound applies one round of the Poseidon permutation.
// State is in Montgomery form throughout.
// scratch is a reusable buffer of length 3*PosT for zero-allocation operation.
func poseidonRound(state, scratch []uint32, r int) {
	// Add round constants (both in Montgomery form, addition preserves form)
	for i := 0; i < field.PosT; i++ {
		state[i] = field.Add(state[i], PosRCsMont[field.PosT*r+i])
	}

	// S-box: x -> x^(-1) in Montgomery form
	// BatchInvMontTree uses tree-based algorithm for O(log n) depth
	// enabling better instruction-level parallelism
	// Note: state elements could be zero after adding round constants
	field.BatchInvMontTreeCond(state, scratch)

	// MDS matrix multiplication: M_ij = 1/(i+j+1)
	// Lazy reduction: accumulate products in uint64, reduce once per row
	// Use scratch as 'old' buffer (reuse after BatchInvMont is done with it)
	// 3 independent accumulator chains for ILP (s01, s23, s4)
	copy(scratch, state)
	scratchArr := (*[field.PosT]uint32)(scratch)
	for i := 0; i < field.PosT; i++ {
		var s01, s23, s4 uint64
		inv := (*[field.PosT]uint32)(PosInvMont[i : i+field.PosT])

		// Fully unroll 35 elements as 7 groups of 5
		// Group 0: j=0..4
		s01 += uint64(inv[0])*uint64(scratchArr[0]) + uint64(inv[1])*uint64(scratchArr[1])
		s23 += uint64(inv[2])*uint64(scratchArr[2]) + uint64(inv[3])*uint64(scratchArr[3])
		s4 += uint64(inv[4]) * uint64(scratchArr[4])

		// Group 1: j=5..9
		s01 += uint64(inv[5])*uint64(scratchArr[5]) + uint64(inv[6])*uint64(scratchArr[6])
		s23 += uint64(inv[7])*uint64(scratchArr[7]) + uint64(inv[8])*uint64(scratchArr[8])
		s4 += uint64(inv[9]) * uint64(scratchArr[9])

		// Group 2: j=10..14
		s01 += uint64(inv[10])*uint64(scratchArr[10]) + uint64(inv[11])*uint64(scratchArr[11])
		s23 += uint64(inv[12])*uint64(scratchArr[12]) + uint64(inv[13])*uint64(scratchArr[13])
		s4 += uint64(inv[14]) * uint64(scratchArr[14])

		// Group 3: j=15..19
		s01 += uint64(inv[15])*uint64(scratchArr[15]) + uint64(inv[16])*uint64(scratchArr[16])
		s23 += uint64(inv[17])*uint64(scratchArr[17]) + uint64(inv[18])*uint64(scratchArr[18])
		s4 += uint64(inv[19]) * uint64(scratchArr[19])

		// Group 4: j=20..24
		s01 += uint64(inv[20])*uint64(scratchArr[20]) + uint64(inv[21])*uint64(scratchArr[21])
		s23 += uint64(inv[22])*uint64(scratchArr[22]) + uint64(inv[23])*uint64(scratchArr[23])
		s4 += uint64(inv[24]) * uint64(scratchArr[24])

		// Group 5: j=25..29
		s01 += uint64(inv[25])*uint64(scratchArr[25]) + uint64(inv[26])*uint64(scratchArr[26])
		s23 += uint64(inv[27])*uint64(scratchArr[27]) + uint64(inv[28])*uint64(scratchArr[28])
		s4 += uint64(inv[29]) * uint64(scratchArr[29])

		// Group 6: j=30..34
		s01 += uint64(inv[30])*uint64(scratchArr[30]) + uint64(inv[31])*uint64(scratchArr[31])
		s23 += uint64(inv[32])*uint64(scratchArr[32]) + uint64(inv[33])*uint64(scratchArr[33])
		s4 += uint64(inv[34]) * uint64(scratchArr[34])

		state[i] = field.MontReduce(s01 + s23 + s4)
	}
}

// PoseidonPerm applies the full Poseidon permutation to state in place.
// State must be in Montgomery form.
func PoseidonPerm(state []uint32) {
	// Tree-based inversion needs n + n/2 + n/4 + ... ≈ 2n scratch
	// For n=35: 35+18+9+5+3+2+1 = 73, so 3*n is safe
	var scratch [3 * field.PosT]uint32
	for r := 0; r < field.PosRF; r++ {
		poseidonRound(state, scratch[:], r)
	}
}

// Poseidon is a sponge construction using the Poseidon permutation.
// Internal state is kept in Montgomery form.
type Poseidon struct {
	s         [field.PosT]uint32     // Montgomery form
	scratch   [3 * field.PosT]uint32 // Reusable scratch buffer for tree-based inversion
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

// perm applies the Poseidon permutation using the internal scratch buffer.
func (p *Poseidon) perm() {
	for r := 0; r < field.PosRF; r++ {
		poseidonRound(p.s[:], p.scratch[:], r)
	}
}

// Write absorbs field elements into the sponge.
// Input is in normal form, converted to Montgomery form internally.
func (p *Poseidon) Write(fes []uint32) {
	if !p.absorbing {
		panic("cannot write after reading")
	}
	for _, fe := range fes {
		// Convert input to Montgomery form and add to state
		feM := field.ToMont(fe)
		p.s[p.i] = field.Add(p.s[p.i], feM)
		p.i++
		if p.i == field.PosRate {
			p.perm()
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
		p.perm()
		p.i = 0
	}
}

// Read squeezes n field elements from the sponge.
// Output is converted from Montgomery form to normal form.
func (p *Poseidon) Read(n int) []uint32 {
	if p.absorbing {
		p.absorbing = false
		if p.i != 0 {
			p.perm()
			p.i = 0
		}
	}

	ret := make([]uint32, 0, n)
	for n > 0 {
		toRead := n
		if toRead > field.PosRate-p.i {
			toRead = field.PosRate - p.i
		}
		// Convert from Montgomery form for output
		for j := 0; j < toRead; j++ {
			ret = append(ret, field.FromMont(p.s[p.i+j]))
		}
		n -= toRead
		p.i += toRead
		if p.i == field.PosRate {
			p.i = 0
			p.perm()
		}
	}
	return ret
}

// ReadNoMod reads n elements from state without modifying position.
// Used in sampleInBall. Returns values in Montgomery form.
func (p *Poseidon) ReadNoMod(n int) []uint32 {
	if n > field.PosRate {
		panic("ReadNoMod: n > PosRate")
	}
	// Return Montgomery form values (sampleInBall expects this now)
	return p.s[:n]
}

// State returns a pointer to the internal state (for sampleInBall).
// State is in Montgomery form.
func (p *Poseidon) State() *[field.PosT]uint32 {
	return &p.s
}

// ApplyPerm applies the Poseidon permutation to the internal state.
func (p *Poseidon) ApplyPerm() {
	p.perm()
}
