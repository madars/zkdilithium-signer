// Package ntt provides Number Theoretic Transform for zkDilithium.
package ntt

import "zkdilithium-signer/pkg/field"

// ZetasMont contains precomputed powers of zeta in Montgomery form.
// ZetasMont[i] = zeta^(brv(i+1)) * R mod Q
// When multiplied with a normal coefficient using MulMont, the result is normal.
var ZetasMont [field.N]uint32

// InvZetasMont contains precomputed powers of inverse zeta in Montgomery form.
var InvZetasMont [field.N]uint32

// Inv2Mont is Inv2 in Montgomery form.
var Inv2Mont uint32

func init() {
	// Compute Montgomery-form zetas
	for i := 0; i < field.N; i++ {
		z := field.Exp(field.Zeta, uint32(field.Brv(uint8(i+1))))
		ZetasMont[i] = field.ToMont(z)
	}

	// Compute Montgomery-form inverse zetas
	for i := 0; i < field.N; i++ {
		exp := 256 - int(field.Brv(uint8(255-i)))
		iz := field.Exp(field.InvZeta, uint32(exp))
		InvZetasMont[i] = field.ToMont(iz)
	}

	// Inv2 in Montgomery form
	Inv2Mont = field.ToMont(field.Inv2)
}

// NTT computes the Number Theoretic Transform of a polynomial in place.
// Input: coefficients in standard order (normal form).
// Output: coefficients in NTT domain (normal form).
// Uses Montgomery multiplication for efficiency.
func NTT(cs *[field.N]uint32) {
	layer := field.N / 2
	zi := 0
	for layer >= 1 {
		for offset := 0; offset < field.N-layer; offset += 2 * layer {
			z := ZetasMont[zi]
			zi++

			for j := offset; j < offset+layer; j++ {
				// MulMont(z_M, c) = z * c (normal form)
				t := field.MulMont(z, cs[j+layer])
				cs[j+layer] = field.Sub(cs[j], t)
				cs[j] = field.Add(cs[j], t)
			}
		}
		layer /= 2
	}
}

// InvNTT computes the inverse Number Theoretic Transform in place.
// Input: coefficients in NTT domain (normal form).
// Output: coefficients in standard order (normal form).
// Uses Montgomery multiplication for efficiency.
func InvNTT(cs *[field.N]uint32) {
	layer := 1
	zi := 0
	for layer < field.N {
		for offset := 0; offset < field.N-layer; offset += 2 * layer {
			z := InvZetasMont[zi]
			zi++

			for j := offset; j < offset+layer; j++ {
				t := field.Sub(cs[j], cs[j+layer])
				// MulMont(Inv2_M, sum) where sum is normal = Inv2 * sum (normal)
				cs[j] = field.MulMont(Inv2Mont, field.Add(cs[j], cs[j+layer]))
				// MulMont(Inv2_M, z_M) where both are Montgomery = (Inv2 * z)_M
				// MulMont((Inv2*z)_M, t) where t is normal = Inv2 * z * t (normal)
				inv2zMont := field.MulMont(Inv2Mont, z)
				cs[j+layer] = field.MulMont(inv2zMont, t)
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
