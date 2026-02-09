package ntt

import (
	"testing"
	"zkdilithium-signer/pkg/field"
)

// Test Zetas values are correct
func TestZetasFirst16(t *testing.T) {
	expected := []uint32{
		2306278, 2001861, 3926523, 5712452, 1922517, 5680261, 4961214, 7026628,
		3353052, 3414003, 1291800, 3770003, 2188519, 44983, 6616885, 4899906,
	}
	for i, want := range expected {
		if Zetas[i] != want {
			t.Errorf("Zetas[%d] = %d, want %d", i, Zetas[i], want)
		}
	}
}

// Test InvZetas values are correct
func TestInvZetasFirst16(t *testing.T) {
	expected := []uint32{
		3141965, 4642089, 4848144, 7181330, 1276293, 6226173, 6371478, 1545565,
		5830703, 4663853, 2915060, 2998944, 5640911, 2250107, 6697852, 5413710,
	}
	for i, want := range expected {
		if InvZetas[i] != want {
			t.Errorf("InvZetas[%d] = %d, want %d", i, InvZetas[i], want)
		}
	}
}

// Test Zetas are computed correctly (catches hardcoding)
func TestZetasComputed(t *testing.T) {
	for i := 0; i < field.N; i++ {
		expected := field.Exp(field.Zeta, uint32(field.Brv(uint8(i+1))))
		if Zetas[i] != expected {
			t.Errorf("Zetas[%d] = %d, want %d", i, Zetas[i], expected)
		}
	}
}

// Test NTT of [1, 0, 0, ...] should be [1, 1, 1, ...]
func TestNTTOfOne(t *testing.T) {
	var cs [field.N]uint32
	cs[0] = 1

	NTT(&cs)

	for i := 0; i < field.N; i++ {
		if cs[i] != 1 {
			t.Errorf("NTT([1,0,...])[%d] = %d, want 1", i, cs[i])
		}
	}
}

// Test NTT(range(256)) first 16 values match Python
func TestNTTOfRangeFirst16(t *testing.T) {
	var cs [field.N]uint32
	for i := 0; i < field.N; i++ {
		cs[i] = uint32(i)
	}

	NTT(&cs)

	expected := []uint32{
		2754782, 330900, 3925693, 7072021, 2466426, 6834207, 295586, 3288141,
		173314, 532343, 1598161, 7075758, 3213908, 3140407, 336540, 5680828,
	}
	for i, want := range expected {
		if cs[i] != want {
			t.Errorf("NTT(range)[%d] = %d, want %d", i, cs[i], want)
		}
	}
}

// Test NTT(range(256)) last 16 values match Python
func TestNTTOfRangeLast16(t *testing.T) {
	var cs [field.N]uint32
	for i := 0; i < field.N; i++ {
		cs[i] = uint32(i)
	}

	NTT(&cs)

	expected := []uint32{
		1985751, 2796981, 2604241, 4522247, 1647302, 1983575, 6357638, 1452416,
		542738, 3830297, 4720115, 1538039, 4911108, 5764199, 1590910, 2423889,
	}
	for i, want := range expected {
		if cs[field.N-16+i] != want {
			t.Errorf("NTT(range)[%d] = %d, want %d", field.N-16+i, cs[field.N-16+i], want)
		}
	}
}

// Test NTT -> InvNTT roundtrip (catches incorrect implementations)
func TestNTTRoundtrip(t *testing.T) {
	var original [field.N]uint32
	var cs [field.N]uint32
	for i := 0; i < field.N; i++ {
		original[i] = uint32(i)
		cs[i] = uint32(i)
	}

	NTT(&cs)
	InvNTT(&cs)

	for i := 0; i < field.N; i++ {
		if cs[i] != original[i] {
			t.Errorf("Roundtrip failed at [%d]: got %d, want %d", i, cs[i], original[i])
		}
	}
}

// Test InvNTT -> NTT roundtrip
func TestInvNTTRoundtrip(t *testing.T) {
	var original [field.N]uint32
	var cs [field.N]uint32
	for i := 0; i < field.N; i++ {
		original[i] = uint32(i)
		cs[i] = uint32(i)
	}

	InvNTT(&cs)
	NTT(&cs)

	for i := 0; i < field.N; i++ {
		if cs[i] != original[i] {
			t.Errorf("InvNTT roundtrip failed at [%d]: got %d, want %d", i, cs[i], original[i])
		}
	}
}

// Test NTT linearity: NTT(a + b) = NTT(a) + NTT(b)
func TestNTTLinearity(t *testing.T) {
	var a, b, sum [field.N]uint32
	var nttA, nttB [field.N]uint32

	for i := 0; i < field.N; i++ {
		a[i] = uint32(i % field.Q)
		b[i] = uint32((2 * i) % field.Q)
		sum[i] = field.Add(a[i], b[i])
		nttA[i] = a[i]
		nttB[i] = b[i]
	}

	NTT(&sum)
	NTT(&nttA)
	NTT(&nttB)

	for i := 0; i < field.N; i++ {
		sumNTT := field.Add(nttA[i], nttB[i])
		if sum[i] != sumNTT {
			t.Errorf("NTT not linear at [%d]: NTT(a+b)=%d, NTT(a)+NTT(b)=%d", i, sum[i], sumNTT)
		}
	}
}

// Benchmark NTT
func BenchmarkNTT(b *testing.B) {
	var cs [field.N]uint32
	for i := 0; i < field.N; i++ {
		cs[i] = uint32(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NTT(&cs)
	}
}

// Benchmark InvNTT
func BenchmarkInvNTT(b *testing.B) {
	var cs [field.N]uint32
	for i := 0; i < field.N; i++ {
		cs[i] = uint32(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InvNTT(&cs)
	}
}
