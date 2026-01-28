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
	field.BatchInvMontTree(state, scratch)

	// MDS matrix multiplication: M_ij = 1/(i+j+1)
	// Lazy reduction: accumulate products in uint64, reduce once per row
	// Use scratch as 'old' buffer (reuse after BatchInvMont is done with it)
	copy(scratch, state)
	scratchArr := (*[field.PosT]uint32)(scratch)
	for i := 0; i < field.PosT; i++ {
		var acc uint64
		invSlice := (*[field.PosT]uint32)(PosInvMont[i : i+field.PosT])
		// Unroll by 5 (35 = 7 × 5) - benchmarked faster than 7-unroll on ARM64
		for j := 0; j < 35; j += 5 {
			t0 := uint64(invSlice[j]) * uint64(scratchArr[j])
			t1 := uint64(invSlice[j+1]) * uint64(scratchArr[j+1])
			t2 := uint64(invSlice[j+2]) * uint64(scratchArr[j+2])
			t3 := uint64(invSlice[j+3]) * uint64(scratchArr[j+3])
			t4 := uint64(invSlice[j+4]) * uint64(scratchArr[j+4])
			acc += t0 + t1 + t2 + t3 + t4
		}
		state[i] = field.MontReduce(acc)
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
