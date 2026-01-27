// Package sampling provides sampling functions for zkDilithium.
package sampling

import (
	"zkdilithium-signer/pkg/encoding"
	"zkdilithium-signer/pkg/field"
	"zkdilithium-signer/pkg/hash"
	"zkdilithium-signer/pkg/poly"
)

// SampleUniform samples a polynomial with uniform coefficients from a byte stream.
func SampleUniform(stream []byte) poly.Poly {
	var cs poly.Poly
	idx := 0
	i := 0
	for i < field.N {
		if idx+3 > len(stream) {
			panic("stream too short")
		}
		d := (uint32(stream[idx]) + (uint32(stream[idx+1]) << 8) + (uint32(stream[idx+2]) << 16)) & 0x7FFFFF
		idx += 3
		if d < field.Q {
			cs[i] = d
			i++
		}
	}
	return cs
}

// SampleLeqEta samples a polynomial with coefficients in [-Eta, Eta] from a byte stream.
func SampleLeqEta(stream []byte) poly.Poly {
	var cs poly.Poly
	idx := 0
	i := 0
	for i < field.N {
		if idx+3 > len(stream) {
			panic("stream too short")
		}
		ds := []uint8{
			stream[idx] & 15,
			stream[idx] >> 4,
			stream[idx+1] & 15,
			stream[idx+1] >> 4,
			stream[idx+2] & 15,
			stream[idx+2] >> 4,
		}
		idx += 3
		for _, d := range ds {
			if d <= 14 {
				// (2 - (d % 5)) mod Q
				cs[i] = field.Mod(int64(2 - int(d%5)))
				i++
				if i >= field.N {
					break
				}
			}
		}
	}
	return cs
}

// SampleMatrix samples the public matrix A from seed rho.
func SampleMatrix(rho []byte) [field.K][field.L]poly.Poly {
	var A [field.K][field.L]poly.Poly
	for i := 0; i < field.K; i++ {
		for j := 0; j < field.L; j++ {
			stream := hash.XOF128(rho, uint16(256*i+j))
			A[i][j] = SampleUniform(stream)
		}
	}
	return A
}

// SampleSecret samples secret vectors s1, s2 from seed rho.
func SampleSecret(rho []byte) (s1 [field.L]poly.Poly, s2 [field.K]poly.Poly) {
	for i := 0; i < field.L; i++ {
		stream := hash.XOF256(rho, uint16(i))
		s1[i] = SampleLeqEta(stream)
	}
	for i := 0; i < field.K; i++ {
		stream := hash.XOF256(rho, uint16(field.L+i))
		s2[i] = SampleLeqEta(stream)
	}
	return
}

// SampleY samples the masking vector y from rho and nonce.
func SampleY(rho []byte, nonce int) [field.L]poly.Poly {
	var y [field.L]poly.Poly
	for i := 0; i < field.L; i++ {
		stream := hash.H(append(rho, byte((nonce+i)&0xFF), byte((nonce+i)>>8)), field.PolyLeGamma1Size)
		y[i] = encoding.UnpackPolyLeGamma1(stream)
	}
	return y
}

// SampleInBall samples the challenge polynomial c from Poseidon hash state.
// Returns nil if rejection sampling fails.
func SampleInBall(h *hash.Poseidon) *poly.Poly {
	var ret poly.Poly
	signsPerFe := uint32(8)
	nTau := ((field.Tau + field.PosCycleLen - 1) / field.PosCycleLen) * field.PosCycleLen
	numCycles := (field.Tau + field.PosCycleLen - 1) / field.PosCycleLen

	for i := 0; i < numCycles; i++ {
		// Apply permutation to internal state
		h.ApplyPerm()
		state := h.State()

		// Read signs from state[8]
		fe := state[8]
		q := fe / (1 << signsPerFe)
		r := fe % (1 << signsPerFe)
		if q == field.Q/(1<<signsPerFe) {
			return nil
		}

		var signs [8]uint32
		for j := 0; j < field.PosCycleLen; j++ {
			if r&1 == 0 {
				signs[j] = 1
			} else {
				signs[j] = field.Q - 1
			}
			r >>= 1
		}

		// Read swap positions from state[0:8]
		for j := 0; j < field.PosCycleLen; j++ {
			base := 256 - nTau + i*field.PosCycleLen + j
			fe := state[j]
			divisor := uint32(base + 1)
			q := fe / divisor
			swapR := int(fe % divisor)
			if q == field.Q/divisor {
				return nil
			}

			ret[base] = ret[swapR]
			ret[swapR] = signs[j]
		}
	}

	return &ret
}
