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
// Inputs and output are in normal (plain) form.
func MulNTT(a, b *Poly, result *Poly) {
	for i := 0; i < field.N; i++ {
		result[i] = field.Mul(a[i], b[i])
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

// BuggyNorm computes the norm with the same bug as the original Python spec.
// The bug: negative coefficients (values > Q/2) are not taken as absolute values,
// effectively ignoring them in the max computation.
// This is needed for compatibility with the Rust prover that uses buggy Python witness.
func (p *Poly) BuggyNorm() uint32 {
	var n uint32
	half := uint32((field.Q - 1) / 2)
	for _, c := range p {
		// Only consider positive values (c <= half).
		// Negative values (c > half) are ignored due to the Python bug
		// where max(c, n) with negative c doesn't update n.
		if c <= half && c > n {
			n = c
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

// DotNTTLazy computes the dot product of L polynomials in NTT domain.
// All inputs must be in NTT domain, normal (plain) form.
// Uses lazy accumulation in uint64 with a single mod-Q reduction per coefficient.
//
// Computes: result[k] = Σ_j (a[j][k] * b[j][k]) mod Q in normal form.
func DotNTTLazy(a, b *[field.L]Poly, result *Poly) {
	for k := 0; k < field.N; k++ {
		acc := uint64(a[0][k]) * uint64(b[0][k])
		acc += uint64(a[1][k]) * uint64(b[1][k])
		acc += uint64(a[2][k]) * uint64(b[2][k])
		acc += uint64(a[3][k]) * uint64(b[3][k])
		result[k] = uint32(acc % field.Q)
	}
}

// MatVecMulNTTLazy computes matrix-vector product A * v in NTT domain.
// A is K×L matrix, v is L-element vector, result is K-element vector.
// All inputs must be in NTT domain, normal (plain) form.
// Uses lazy accumulation for better performance.
func MatVecMulNTTLazy(A *[field.K][field.L]Poly, v *[field.L]Poly, result *[field.K]Poly) {
	for i := 0; i < field.K; i++ {
		DotNTTLazy(&A[i], v, &result[i])
	}
}

// Copy copies src to dst.
func Copy(dst, src *Poly) {
	*dst = *src
}
