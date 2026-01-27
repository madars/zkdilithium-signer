package hash

import "zkdilithium-signer/pkg/field"

// Grain is an LFSR used to generate MDS and round constants for Poseidon.
// Uses two uint64s to store an 80-bit state.
type Grain struct {
	lo uint64 // bits 0-63
	hi uint64 // bits 64-79 (only lower 16 bits used)
}

// NewGrain creates a new Grain LFSR initialized for Poseidon constant generation.
func NewGrain() *Grain {
	g := &Grain{}

	// Initialize 80-bit state per Poseidon spec:
	// state = (2^30-1) | (0 << 30) | (POS_RF << 40) | (POS_T << 50) | (POS_RATE << 62) | (2 << 74) | (1 << 78)

	// Lower 64 bits (bits 0-63)
	g.lo = (1 << 30) - 1                                     // bits 0-29: 2^30-1
	g.lo |= 0 << 30                                          // bits 30-39: 0 partial rounds
	g.lo |= uint64(field.PosRF) << 40                        // bits 40-49: full rounds (21)
	g.lo |= uint64(field.PosT) << 50                         // bits 50-61: state size (35)
	g.lo |= uint64(field.PosRate&0x3) << 62                  // bits 62-63: lower 2 bits of rate

	// Upper 16 bits (bits 64-79)
	g.hi = uint64(field.PosRate >> 2)                        // bits 64-73: upper bits of rate
	g.hi |= 2 << 10                                          // bits 74-77: alpha = -1 encoded as 2
	g.hi |= 1 << 14                                          // bit 78: odd Q

	// Discard first 160 bits
	for i := 0; i < 160; i++ {
		g.next()
	}
	return g
}

// getBit returns bit i of the 80-bit state.
func (g *Grain) getBit(i int) uint64 {
	if i < 64 {
		return (g.lo >> i) & 1
	}
	return (g.hi >> (i - 64)) & 1
}

// next returns the next bit from the LFSR.
func (g *Grain) next() uint8 {
	// Tap positions: 17, 28, 41, 56, 66, 79
	r := g.getBit(17) ^ g.getBit(28) ^ g.getBit(41) ^ g.getBit(56) ^ g.getBit(66) ^ g.getBit(79)

	// Shift state left by 1, insert r at position 0
	// Carry from lo to hi
	carry := (g.lo >> 63) & 1
	g.lo = (g.lo << 1) | r
	g.hi = ((g.hi << 1) | carry) & 0xFFFF // keep only 16 bits

	return uint8(r)
}

// ReadBits reads n bits from the LFSR.
func (g *Grain) ReadBits(n int) uint32 {
	var got int
	var ret uint32
	for got < n {
		first := g.next()
		second := g.next()
		if first == 1 {
			ret = (ret << 1) | uint32(second)
			got++
		}
	}
	return ret
}

// ReadFe reads a field element from the LFSR (rejection sampling).
func (g *Grain) ReadFe() uint32 {
	for {
		x := g.ReadBits(23)
		if x < field.Q {
			return x
		}
	}
}
