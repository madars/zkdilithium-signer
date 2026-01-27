// Package ntt provides Number Theoretic Transform for zkDilithium.
package ntt

import "zkdilithium-signer/pkg/field"

// Zetas contains precomputed powers of zeta with bit-reversed indices.
// Zetas[i] = zeta^(brv(i+1)) mod Q
var Zetas [field.N]uint32

// InvZetas contains precomputed powers of inverse zeta.
// InvZetas[i] = invzeta^(256-brv(255-i)) mod Q
var InvZetas [field.N]uint32

func init() {
	// Compute Zetas
	for i := 0; i < field.N; i++ {
		Zetas[i] = field.Exp(field.Zeta, uint32(field.Brv(uint8(i+1))))
	}

	// Compute InvZetas
	for i := 0; i < field.N; i++ {
		exp := 256 - int(field.Brv(uint8(255-i)))
		InvZetas[i] = field.Exp(field.InvZeta, uint32(exp))
	}
}

// NTT computes the Number Theoretic Transform of a polynomial in place.
// Input: coefficients in standard order.
// Output: coefficients in NTT domain.
func NTT(cs *[field.N]uint32) {
	layer := field.N / 2
	zi := 0
	for layer >= 1 {
		for offset := 0; offset < field.N-layer; offset += 2 * layer {
			z := Zetas[zi]
			zi++

			for j := offset; j < offset+layer; j++ {
				t := field.Mul(z, cs[j+layer])
				cs[j+layer] = field.Sub(cs[j], t)
				cs[j] = field.Add(cs[j], t)
			}
		}
		layer /= 2
	}
}

// InvNTT computes the inverse Number Theoretic Transform in place.
// Input: coefficients in NTT domain.
// Output: coefficients in standard order.
func InvNTT(cs *[field.N]uint32) {
	layer := 1
	zi := 0
	for layer < field.N {
		for offset := 0; offset < field.N-layer; offset += 2 * layer {
			z := InvZetas[zi]
			zi++

			for j := offset; j < offset+layer; j++ {
				t := field.Sub(cs[j], cs[j+layer])
				cs[j] = field.Mul(field.Inv2, field.Add(cs[j], cs[j+layer]))
				cs[j+layer] = field.Mul(field.Mul(field.Inv2, z), t)
			}
		}
		layer *= 2
	}
}

// MulNTT performs componentwise multiplication of two polynomials in NTT domain.
func MulNTT(a, b *[field.N]uint32, result *[field.N]uint32) {
	for i := 0; i < field.N; i++ {
		result[i] = field.Mul(a[i], b[i])
	}
}
