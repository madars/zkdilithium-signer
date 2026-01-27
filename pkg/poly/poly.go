// Package poly provides polynomial operations for zkDilithium.
package poly

import (
	"zkdilithium-signer/pkg/field"
	"zkdilithium-signer/pkg/ntt"
)

// Poly represents a polynomial in Z_Q[x]/<x^256+1>.
type Poly [field.N]uint32

// Add computes a + b componentwise.
func Add(a, b *Poly, result *Poly) {
	for i := 0; i < field.N; i++ {
		result[i] = field.Add(a[i], b[i])
	}
}

// Sub computes a - b componentwise.
func Sub(a, b *Poly, result *Poly) {
	for i := 0; i < field.N; i++ {
		result[i] = field.Sub(a[i], b[i])
	}
}

// Neg computes -a componentwise.
func Neg(a *Poly, result *Poly) {
	for i := 0; i < field.N; i++ {
		result[i] = field.Neg(a[i])
	}
}

// NTT computes NTT in place.
func (p *Poly) NTT() {
	ntt.NTT((*[field.N]uint32)(p))
}

// InvNTT computes inverse NTT in place.
func (p *Poly) InvNTT() {
	ntt.InvNTT((*[field.N]uint32)(p))
}

// MulNTT computes componentwise multiplication (for polynomials in NTT domain).
// Uses Montgomery multiplication - both inputs should be in Montgomery form,
// and the result will be in Montgomery form.
func MulNTT(a, b *Poly, result *Poly) {
	for i := 0; i < field.N; i++ {
		result[i] = field.MulMont(a[i], b[i])
	}
}

// ToMont converts polynomial to Montgomery form in place.
func (p *Poly) ToMont() {
	for i := 0; i < field.N; i++ {
		p[i] = field.ToMont(p[i])
	}
}

// FromMont converts polynomial from Montgomery form in place.
func (p *Poly) FromMont() {
	for i := 0; i < field.N; i++ {
		p[i] = field.FromMont(p[i])
	}
}

// SchoolbookMul computes a * b using schoolbook multiplication.
// Returns (quotient, remainder) where a * b = quotient * (x^256 + 1) + remainder.
func SchoolbookMul(a, b *Poly) (q, r Poly) {
	// Compute full product (511 coefficients, last is 0)
	var s [512]int64
	for i := 0; i < field.N; i++ {
		for j := 0; j < field.N; j++ {
			s[i+j] += int64(a[i]) * int64(b[j])
		}
	}

	// Reduce coefficients mod Q
	for i := 0; i < 511; i++ {
		s[i] = s[i] % field.Q
		if s[i] < 0 {
			s[i] += field.Q
		}
	}

	// quotient is coefficients [256:512]
	for i := 0; i < field.N; i++ {
		q[i] = uint32(s[256+i])
	}

	// remainder is coefficients [0:256] - coefficients [256:512] (because x^256 = -1)
	for i := 0; i < field.N; i++ {
		r[i] = field.Mod(s[i] - s[256+i])
	}

	return q, r
}

// Norm returns the infinity norm (max absolute coefficient).
// For signed interpretation: values > Q/2 are treated as negative.
func (p *Poly) Norm() uint32 {
	var n uint32
	half := uint32((field.Q - 1) / 2)
	for _, c := range p {
		var absC uint32
		if c > half {
			absC = field.Q - c
		} else {
			absC = c
		}
		if absC > n {
			n = absC
		}
	}
	return n
}

// Decompose decomposes each coefficient using field.Decompose.
func (p *Poly) Decompose() (p0 Poly, p1 Poly) {
	for i := 0; i < field.N; i++ {
		r0, r1 := field.Decompose(p[i])
		p0[i] = field.Mod(int64(r0))
		p1[i] = r1
	}
	return
}


// Equal returns true if two polynomials are equal.
func Equal(a, b *Poly) bool {
	for i := 0; i < field.N; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Copy copies src to dst.
func Copy(dst, src *Poly) {
	*dst = *src
}
