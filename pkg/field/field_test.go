package field

import "testing"

// Test constants match Python implementation
func TestConstants(t *testing.T) {
	if Q != 7340033 {
		t.Errorf("Q = %d, want 7340033", Q)
	}
	if N != 256 {
		t.Errorf("N = %d, want 256", N)
	}
	if Zeta != 3483618 {
		t.Errorf("Zeta = %d, want 3483618", Zeta)
	}
	if InvZeta != 3141965 {
		t.Errorf("InvZeta = %d, want 3141965", InvZeta)
	}
	if Inv2 != 3670017 {
		t.Errorf("Inv2 = %d, want 3670017", Inv2)
	}
	if Gamma1 != 131072 {
		t.Errorf("Gamma1 = %d, want 131072", Gamma1)
	}
	if Gamma2 != 65536 {
		t.Errorf("Gamma2 = %d, want 65536", Gamma2)
	}
	if Eta != 2 {
		t.Errorf("Eta = %d, want 2", Eta)
	}
	if K != 4 || L != 4 {
		t.Errorf("K=%d, L=%d, want 4, 4", K, L)
	}
	if Tau != 40 {
		t.Errorf("Tau = %d, want 40", Tau)
	}
	if Beta != 80 {
		t.Errorf("Beta = %d, want 80", Beta)
	}
}

// Test modular inverse with known values from Python
func TestInv(t *testing.T) {
	tests := []struct {
		input, want uint32
	}{
		{1, 1},
		{2, 3670017},
		{3, 2446678},
		{1000, 2224030},
		{Q - 1, 7340032},
		{123456, 2165041},
	}
	for _, tc := range tests {
		got := Inv(tc.input)
		if got != tc.want {
			t.Errorf("Inv(%d) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

// Test that Inv actually computes inverse (catches hardcoded returns)
func TestInvProperty(t *testing.T) {
	testCases := []uint32{1, 2, 3, 7, 13, 1000, 123456, Q - 1, Q - 2, Q / 2}
	for _, a := range testCases {
		aInv := Inv(a)
		product := Mul(a, aInv)
		if product != 1 {
			t.Errorf("Inv(%d) = %d, but %d * %d = %d (want 1)", a, aInv, a, aInv, product)
		}
	}
}

// Test that optimized Inv matches generic Exp(a, Q-2) method
func TestInvMatchesExp(t *testing.T) {
	testCases := []uint32{1, 2, 3, 7, 13, 42, 100, 1000, 12345, 123456, 1000000, Q - 1, Q - 2, Q / 2, Q / 3}
	for _, a := range testCases {
		got := Inv(a)
		want := Exp(a, Q-2)
		if got != want {
			t.Errorf("Inv(%d) = %d, but Exp(%d, Q-2) = %d", a, got, a, want)
		}
	}
}

// Test Inv(0) returns 0 (undefined but safe)
func TestInvZero(t *testing.T) {
	if Inv(0) != 0 {
		t.Errorf("Inv(0) = %d, want 0", Inv(0))
	}
}

// Test that Inv2 is actually inverse of 2
func TestInv2Property(t *testing.T) {
	if Mul(2, Inv2) != 1 {
		t.Errorf("2 * Inv2 = %d, want 1", Mul(2, Inv2))
	}
}

// Test that InvZeta is actually inverse of Zeta
func TestInvZetaProperty(t *testing.T) {
	if Mul(Zeta, InvZeta) != 1 {
		t.Errorf("Zeta * InvZeta = %d, want 1", Mul(Zeta, InvZeta))
	}
}

// Test that Zeta is a 512th primitive root of unity
func TestZetaPrimitiveRoot(t *testing.T) {
	// Zeta^256 should equal -1 (mod Q) = Q-1
	z256 := Exp(Zeta, 256)
	if z256 != Q-1 {
		t.Errorf("Zeta^256 = %d, want %d (Q-1)", z256, Q-1)
	}

	// Zeta^512 should equal 1
	z512 := Exp(Zeta, 512)
	if z512 != 1 {
		t.Errorf("Zeta^512 = %d, want 1", z512)
	}

	// Zeta^128 should NOT equal 1 (primitive)
	z128 := Exp(Zeta, 128)
	if z128 == 1 {
		t.Errorf("Zeta^128 = 1, but Zeta should be primitive")
	}
}

// Test bit reversal with known values from Python
func TestBrv(t *testing.T) {
	tests := []struct {
		input, want uint8
	}{
		{0, 0},
		{1, 128},
		{2, 64},
		{127, 254},
		{128, 1},
		{255, 255},
		{170, 85}, // 0b10101010 -> 0b01010101
	}
	for _, tc := range tests {
		got := Brv(tc.input)
		if got != tc.want {
			t.Errorf("Brv(%d) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

// Test bit reversal is involution (catches incorrect implementation)
func TestBrvInvolution(t *testing.T) {
	for i := 0; i < 256; i++ {
		x := uint8(i)
		if Brv(Brv(x)) != x {
			t.Errorf("Brv(Brv(%d)) = %d, want %d", x, Brv(Brv(x)), x)
		}
	}
}

// Test decompose with known values from Python
func TestDecompose(t *testing.T) {
	tests := []struct {
		input uint32
		r0    int32
		r1    uint32
	}{
		{0, 0, 0},
		{1, 1, 0},
		{65536, 65536, 0},  // Gamma2
		{131072, 0, 1},     // 2*Gamma2
		{7340032, -1, 0},   // Q-1
		{7274497, -65536, 0}, // Q-Gamma2
		{3670016, 0, 28},   // Q/2 approx
		{12345, 12345, 0},
		{7327688, -12345, 0}, // Q-12345
	}
	for _, tc := range tests {
		r0, r1 := Decompose(tc.input)
		if r0 != tc.r0 || r1 != tc.r1 {
			t.Errorf("Decompose(%d) = (%d, %d), want (%d, %d)",
				tc.input, r0, r1, tc.r0, tc.r1)
		}
	}
}

// Test decompose reconstruction property (catches cheating)
func TestDecomposeReconstruction(t *testing.T) {
	testCases := []uint32{0, 1, Gamma2, 2 * Gamma2, Q - 1, Q - Gamma2, Q / 2, 12345, Q - 12345}
	for _, r := range testCases {
		r0, r1 := Decompose(r)
		// Reconstruct: r1 * 2 * Gamma2 + r0
		reconstructed := Mod(int64(r1)*2*Gamma2 + int64(r0))
		if reconstructed != r {
			t.Errorf("Decompose(%d) = (%d, %d), reconstructs to %d", r, r0, r1, reconstructed)
		}
	}
}

// Test basic arithmetic
func TestArithmetic(t *testing.T) {
	// Add
	if Add(Q-1, 1) != 0 {
		t.Errorf("Add(Q-1, 1) = %d, want 0", Add(Q-1, 1))
	}
	if Add(Q-1, 2) != 1 {
		t.Errorf("Add(Q-1, 2) = %d, want 1", Add(Q-1, 2))
	}

	// Sub
	if Sub(0, 1) != Q-1 {
		t.Errorf("Sub(0, 1) = %d, want %d", Sub(0, 1), Q-1)
	}
	if Sub(100, 30) != 70 {
		t.Errorf("Sub(100, 30) = %d, want 70", Sub(100, 30))
	}

	// Mul
	if Mul(2, 3) != 6 {
		t.Errorf("Mul(2, 3) = %d, want 6", Mul(2, 3))
	}

	// Neg
	if Neg(1) != Q-1 {
		t.Errorf("Neg(1) = %d, want %d", Neg(1), Q-1)
	}
	if Neg(0) != 0 {
		t.Errorf("Neg(0) = %d, want 0", Neg(0))
	}

	// Mod with negative
	if Mod(-1) != Q-1 {
		t.Errorf("Mod(-1) = %d, want %d", Mod(-1), Q-1)
	}
}

// Test BatchInv correctness
func TestBatchInv(t *testing.T) {
	// Test with known values
	xs := []uint32{1, 2, 3, 1000, 123456}
	expected := make([]uint32, len(xs))
	for i, x := range xs {
		expected[i] = Inv(x)
	}

	BatchInv(xs)

	for i, want := range expected {
		if xs[i] != want {
			t.Errorf("BatchInv[%d] = %d, want %d", i, xs[i], want)
		}
	}
}

// Test BatchInv with zeros (edge case)
func TestBatchInvWithZeros(t *testing.T) {
	xs := []uint32{0, 2, 0, 1000, 0}
	expected := []uint32{0, Inv(2), 0, Inv(1000), 0}

	BatchInv(xs)

	for i, want := range expected {
		if xs[i] != want {
			t.Errorf("BatchInv with zeros [%d] = %d, want %d", i, xs[i], want)
		}
	}
}

// Test BatchInv property: each result is actually the inverse
func TestBatchInvProperty(t *testing.T) {
	original := []uint32{7, 13, 42, 1000, 123456, Q - 1, Q - 2}
	xs := make([]uint32, len(original))
	copy(xs, original)

	BatchInv(xs)

	for i, inv := range xs {
		product := Mul(original[i], inv)
		if product != 1 {
			t.Errorf("BatchInv: %d * %d = %d, want 1", original[i], inv, product)
		}
	}
}

// Test BatchInv with single element
func TestBatchInvSingle(t *testing.T) {
	xs := []uint32{42}
	expected := Inv(42)
	BatchInv(xs)
	if xs[0] != expected {
		t.Errorf("BatchInv single: got %d, want %d", xs[0], expected)
	}
}

// Test BatchInv with empty slice
func TestBatchInvEmpty(t *testing.T) {
	xs := []uint32{}
	BatchInv(xs) // Should not panic
}

// Test BatchInv with PosT elements (actual use case)
func TestBatchInvPosT(t *testing.T) {
	xs := make([]uint32, PosT)
	expected := make([]uint32, PosT)
	for i := 0; i < PosT; i++ {
		xs[i] = uint32(i + 1) // 1, 2, 3, ..., 35
		expected[i] = Inv(xs[i])
	}

	BatchInv(xs)

	for i := 0; i < PosT; i++ {
		if xs[i] != expected[i] {
			t.Errorf("BatchInv PosT [%d] = %d, want %d", i, xs[i], expected[i])
		}
	}
}

