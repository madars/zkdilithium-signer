// Package field provides finite field arithmetic for zkDilithium.
//
// The field is Z_Q where Q = 2^23 - 2^20 + 1 = 7340033.
package field

import "math/bits"

const (
	// Q is the prime modulus: 2^23 - 2^20 + 1
	Q = 7340033

	// N is the polynomial degree (ring is Z_Q[x]/<x^256+1>)
	N = 256

	// NBits is log2(N)
	NBits = 8

	// Zeta is a 512th primitive root of unity in Z_Q
	// Computed as pow(3, (Q-1)/512, Q)
	Zeta = 3483618

	// InvZeta is the modular inverse of Zeta
	InvZeta = 3141965

	// Inv2 is the modular inverse of 2: (Q+1)/2
	Inv2 = 3670017

	// Dilithium parameters
	Gamma1 = 1 << 17 // 131072
	Gamma2 = 65536   // (Q-1)/112
	Eta    = 2
	K      = 4
	L      = 4
	Tau    = 40
	Beta   = Tau * Eta // 80

	// Poseidon parameters
	PosT        = 35
	PosRate     = 24
	PosRF       = 21
	PosCycleLen = 8

	// Signature encoding sizes
	CSize            = 12 // field elements for c tilde
	MuSize           = 24 // field elements for mu
	PolyLeGamma1Size = 576
)

// Mod returns x mod Q, handling negative values correctly.
func Mod(x int64) uint32 {
	x = x % Q
	if x < 0 {
		x += Q
	}
	return uint32(x)
}

// Add returns (a + b) mod Q.
// Since Q ~ 2^23, a + b < 2*Q < 2^24 fits in uint32.
func Add(a, b uint32) uint32 {
	sum := a + b
	if sum >= Q {
		sum -= Q
	}
	return sum
}

// Sub returns (a - b) mod Q.
// Using int32 arithmetic avoids extra comparison.
func Sub(a, b uint32) uint32 {
	diff := int32(a) - int32(b)
	if diff < 0 {
		diff += Q
	}
	return uint32(diff)
}

// Mul returns (a * b) mod Q.
func Mul(a, b uint32) uint32 {
	return uint32((uint64(a) * uint64(b)) % Q)
}

// Neg returns (-a) mod Q = Q - a for a != 0.
func Neg(a uint32) uint32 {
	if a == 0 {
		return 0
	}
	return Q - a
}

// Inv returns the modular inverse of a using Fermat's little theorem: a^(Q-2) mod Q.
// Uses an optimized addition chain exploiting Q-2 = 0b110_11111111111111111111.
// Cost: 30 operations (23 squarings, 7 multiplications) vs ~43 for binary method.
func Inv(a uint32) uint32 {
	if a == 0 {
		return 0 // Undefined, but 0 is safe return
	}

	// Q-2 = 7340031 = 0b110_11111111111111111111
	// Structure: header "110" followed by 20 ones = 5 blocks of "1111"

	// 1. Precompute small powers (3 squarings, 2 multiplications)
	x2 := Mul(a, a)     // a^2
	x3 := Mul(x2, a)    // a^3  (bits: 11)
	x6 := Mul(x3, x3)   // a^6  (bits: 110) <- header
	x12 := Mul(x6, x6)  // a^12
	x15 := Mul(x12, x3) // a^15 (bits: 1111)

	// 2. Append "1111" five times to the header "110"
	// Exponent structure: [110] [1111] [1111] [1111] [1111] [1111]
	res := x6

	// Iteration 1: shift left 4, append 1111
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, x15)

	// Iteration 2
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, x15)

	// Iteration 3
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, x15)

	// Iteration 4
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, x15)

	// Iteration 5
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, res)
	res = Mul(res, x15)

	return res
}

// BatchInv computes the modular inverse of each element in place.
// Uses Montgomery's trick: n inversions with 1 inversion + 3(n-1) multiplications.
// Elements that are 0 remain 0 (since 0^(-1) is undefined, we treat it as 0).
func BatchInv(xs []uint32) {
	n := len(xs)
	if n == 0 {
		return
	}

	// Compute prefix products: prods[i] = xs[0] * xs[1] * ... * xs[i]
	// Skip zeros by treating them as 1 in the product
	prods := make([]uint32, n)
	prods[0] = xs[0]
	if prods[0] == 0 {
		prods[0] = 1
	}
	for i := 1; i < n; i++ {
		if xs[i] == 0 {
			prods[i] = prods[i-1]
		} else {
			prods[i] = Mul(prods[i-1], xs[i])
		}
	}

	// Invert the final product
	inv := Inv(prods[n-1])

	// Work backwards to compute individual inverses
	for i := n - 1; i > 0; i-- {
		if xs[i] == 0 {
			// 0 stays 0
			continue
		}
		// Save original value
		oldXi := xs[i]
		// xs[i]^(-1) = inv * prods[i-1]
		xs[i] = Mul(inv, prods[i-1])
		// Update inv to be prods[i-1]^(-1) = inv * oldXi
		inv = Mul(inv, oldXi)
	}
	// Handle first element
	if xs[0] != 0 {
		xs[0] = inv
	}
}

// Exp returns a^e mod Q using binary exponentiation.
func Exp(a uint32, e uint32) uint32 {
	result := uint64(1)
	base := uint64(a)
	for e > 0 {
		if e&1 == 1 {
			result = (result * base) % Q
		}
		base = (base * base) % Q
		e >>= 1
	}
	return uint32(result)
}

// Brv reverses an 8-bit number (bit reversal for NTT).
func Brv(x uint8) uint8 {
	return bits.Reverse8(x)
}

// Decompose splits r into (r0, r1) such that r = r1 * 2 * Gamma2 + r0.
// r0 is in (-Gamma2, Gamma2] except when r >= Q - Gamma2.
func Decompose(r uint32) (r0 int32, r1 uint32) {
	// r0 = r mod (2*Gamma2)
	r0 = int32(r % (2 * Gamma2))
	if r0 > Gamma2 {
		r0 -= 2 * Gamma2
	}
	if r-uint32(r0) == Q-1 {
		return r0 - 1, 0
	}
	return r0, (r - uint32(r0)) / (2 * Gamma2)
}

const (
	// barrettMu64Floor = floor(2^64 / Q).
	barrettMu64Floor uint64 = ^uint64(0) / Q
)

// reduce brings a value < 2Q back to < Q in constant time (branchless).
// Uses a sign-bit mask to avoid branch misprediction (~50% taken for uniform input).
func reduce(a uint32) uint32 {
	// If a >= Q: (a - Q) is positive, mask = 0x00000000
	// If a <  Q: (a - Q) is negative (int32 view), mask = 0xFFFFFFFF
	b := a - Q
	mask := uint32(int32(b) >> 31)
	// If mask is -1: returns b + Q = a
	// If mask is 0:  returns b = a - Q
	return b + (Q & mask)
}

// reduceBarrett64Lazy computes a lazy representative of p mod Q.
// For p < 4Q^2, output is in [0, 2Q).
func reduceBarrett64Lazy(p uint64) uint32 {
	q, _ := bits.Mul64(p, barrettMu64Floor)
	return uint32(p - q*uint64(Q))
}

// mulPlainLazy computes a*b mod Q in lazy form [0, 2Q).
// Requires a,b < 2Q.
func mulPlainLazy(a, b uint32) uint32 {
	return reduceBarrett64Lazy(uint64(a) * uint64(b))
}

// mulPlainLazy2 computes two independent lazy products.
// It is structured to expose ILP across the two reduction chains.
func mulPlainLazy2(a0, b0, a1, b1 uint32) (r0, r1 uint32) {
	p0 := uint64(a0) * uint64(b0)
	p1 := uint64(a1) * uint64(b1)
	q0, _ := bits.Mul64(p0, barrettMu64Floor)
	q1, _ := bits.Mul64(p1, barrettMu64Floor)
	return uint32(p0 - q0*uint64(Q)), uint32(p1 - q1*uint64(Q))
}

// mulPlainStrict computes canonical a*b mod Q in [0, Q).
func mulPlainStrict(a, b uint32) uint32 {
	return reduce(mulPlainLazy(a, b))
}

// reduce2 canonicalizes two values in [0, 2Q) to [0, Q).
func reduce2(a0, a1 uint32) (r0, r1 uint32) {
	b0 := a0 - Q
	b1 := a1 - Q
	m0 := uint32(int32(b0) >> 31)
	m1 := uint32(int32(b1) >> 31)
	return b0 + (Q & m0), b1 + (Q & m1)
}

// mulPlainStrict2 computes two independent strict products in [0, Q).
func mulPlainStrict2(a0, b0, a1, b1 uint32) (r0, r1 uint32) {
	l0, l1 := mulPlainLazy2(a0, b0, a1, b1)
	if l0 >= Q {
		l0 -= Q
	}
	if l1 >= Q {
		l1 -= Q
	}
	return l0, l1
}

// invPlainLazy computes a^(Q-2) mod Q using an optimized addition chain with plain-domain
// lazy multiplication internally, with a single final canonical reduction.
func invPlainLazy(a uint32) uint32 {
	if a == 0 {
		return 0
	}

	_10 := mulPlainLazy(a, a)
	_11 := mulPlainLazy(a, _10)
	_1100 := mulPlainLazy(_11, _11)
	_1100 = mulPlainLazy(_1100, _1100)
	_1111 := mulPlainLazy(_11, _1100)
	_1100000 := mulPlainLazy(_1100, _1100)
	_1100000 = mulPlainLazy(_1100000, _1100000)
	_1100000 = mulPlainLazy(_1100000, _1100000)
	_1101111 := mulPlainLazy(_1111, _1100000)

	i23 := mulPlainLazy(_1101111, _1101111)
	i23 = mulPlainLazy(i23, i23)
	i23 = mulPlainLazy(i23, i23)
	i23 = mulPlainLazy(i23, i23)
	i23 = mulPlainLazy(i23, _1111)
	i23 = mulPlainLazy(i23, i23)
	i23 = mulPlainLazy(i23, i23)
	i23 = mulPlainLazy(i23, i23)
	i23 = mulPlainLazy(i23, i23)
	i23 = mulPlainLazy(i23, _1111)
	i23 = mulPlainLazy(i23, i23)
	i23 = mulPlainLazy(i23, i23)
	i23 = mulPlainLazy(i23, i23)
	i23 = mulPlainLazy(i23, i23)

	res := mulPlainLazy(_1111, i23)
	res = mulPlainLazy(res, res)
	res = mulPlainLazy(res, res)
	res = mulPlainLazy(res, res)
	res = mulPlainLazy(res, res)
	res = mulPlainLazy(res, _1111)

	return reduce(res)
}

