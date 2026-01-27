// Package field provides finite field arithmetic for zkDilithium.
//
// The field is Z_Q where Q = 2^23 - 2^20 + 1 = 7340033.
package field

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
	PosT       = 35
	PosRate    = 24
	PosRF      = 21
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
func Add(a, b uint32) uint32 {
	sum := uint64(a) + uint64(b)
	if sum >= Q {
		sum -= Q
	}
	return uint32(sum)
}

// Sub returns (a - b) mod Q.
func Sub(a, b uint32) uint32 {
	if a >= b {
		return a - b
	}
	return Q - b + a
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
func Inv(a uint32) uint32 {
	return Exp(a, Q-2)
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
	x = (x&0xF0)>>4 | (x&0x0F)<<4
	x = (x&0xCC)>>2 | (x&0x33)<<2
	x = (x&0xAA)>>1 | (x&0x55)<<1
	return x
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
