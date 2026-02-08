package hash

import (
	"testing"
	"zkdilithium-signer/pkg/field"
)

// Test Grain first 10 field elements match Python
func TestGrainFirst10Fes(t *testing.T) {
	g := NewGrain()
	expected := []uint32{
		662000, 7104925, 2304656, 2330809, 452951,
		1722141, 5334010, 7087604, 5110463, 6023804,
	}
	for i, want := range expected {
		got := g.ReadFe()
		if got != want {
			t.Errorf("Grain.ReadFe()[%d] = %d, want %d", i, got, want)
		}
	}
}

// Test round constants first 16 match Python.
func TestPosRCsFirst16(t *testing.T) {
	expected := []uint32{
		662000, 7104925, 2304656, 2330809, 452951, 1722141, 5334010, 7087604,
		5110463, 6023804, 3061965, 6087945, 3740272, 284272, 4421217, 559188,
	}
	for i, want := range expected {
		got := PosRCs[i]
		if got != want {
			t.Errorf("PosRCs[%d] = %d, want %d", i, got, want)
		}
	}
}

// Test Poseidon permutation with known values from Python
func TestPoseidonPerm(t *testing.T) {
	state := make([]uint32, field.PosT)
	for i := 0; i < field.PosT; i++ {
		state[i] = uint32(i)
	}

	PoseidonPerm(state)

	expected := []uint32{
		6525793, 2817790, 5538989, 1140645, 1838881, 2536727, 6768730, 4709337,
		6955613, 2401101, 1387526, 5346661, 1137806, 7270459, 1552970, 4071298,
		3931520, 4509604, 1434920, 2477273, 4595089, 4960924, 2665912, 5601770,
		3176785, 6236514, 4336216, 2469459, 2737160, 6481909, 5295937, 1830143,
		7322777, 3396423, 2354672,
	}
	for i, want := range expected {
		got := state[i]
		if got != want {
			t.Errorf("PoseidonPerm[%d] = %d, want %d", i, got, want)
		}
	}
}

// Test Poseidon sponge with known values from Python
func TestPoseidonSponge(t *testing.T) {
	h := NewPoseidon([]uint32{1, 2, 3})
	result := h.Read(12)

	expected := []uint32{
		1948781, 4026402, 5373296, 2459025, 3075965, 1506296,
		229209, 7105271, 5926873, 2350085, 6176282, 5229836,
	}
	for i, want := range expected {
		if result[i] != want {
			t.Errorf("Poseidon sponge[%d] = %d, want %d", i, result[i], want)
		}
	}
}

// Test that Grain produces valid field elements (catches incorrect implementation)
func TestGrainProducesValidFes(t *testing.T) {
	g := NewGrain()
	for i := 0; i < 100; i++ {
		fe := g.ReadFe()
		if fe >= field.Q {
			t.Errorf("Grain.ReadFe() = %d >= Q", fe)
		}
	}
}

// Test Poseidon is deterministic (same input -> same output)
func TestPoseidonDeterministic(t *testing.T) {
	h1 := NewPoseidon([]uint32{1, 2, 3})
	h2 := NewPoseidon([]uint32{1, 2, 3})

	r1 := h1.Read(10)
	r2 := h2.Read(10)

	for i := range r1 {
		if r1[i] != r2[i] {
			t.Errorf("Poseidon not deterministic at [%d]: %d != %d", i, r1[i], r2[i])
		}
	}
}

// Test Poseidon different inputs give different outputs
func TestPoseidonDifferentInputs(t *testing.T) {
	h1 := NewPoseidon([]uint32{1, 2, 3})
	h2 := NewPoseidon([]uint32{1, 2, 4})

	r1 := h1.Read(10)
	r2 := h2.Read(10)

	same := true
	for i := range r1 {
		if r1[i] != r2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("Different inputs produced same output")
	}
}

// Benchmark MDS matrix multiplication (current 5-unroll)
func BenchmarkMDS(b *testing.B) {
	state := make([]uint32, field.PosT)
	scratch := make([]uint32, field.PosT)
	for i := 0; i < field.PosT; i++ {
		state[i] = uint32(i + 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(scratch, state)
		scratchArr := (*[field.PosT]uint32)(scratch)
		for row := 0; row < field.PosT; row++ {
			var acc uint64
			invSlice := (*[field.PosT]uint32)(PosInv[row : row+field.PosT])
			for j := 0; j < 35; j += 5 {
				t0 := uint64(invSlice[j]) * uint64(scratchArr[j])
				t1 := uint64(invSlice[j+1]) * uint64(scratchArr[j+1])
				t2 := uint64(invSlice[j+2]) * uint64(scratchArr[j+2])
				t3 := uint64(invSlice[j+3]) * uint64(scratchArr[j+3])
				t4 := uint64(invSlice[j+4]) * uint64(scratchArr[j+4])
				acc += t0 + t1 + t2 + t3 + t4
			}
			state[row] = uint32(acc % field.Q)
		}
	}
}

// Benchmark MDS with 7-unroll
func BenchmarkMDS7Unroll(b *testing.B) {
	state := make([]uint32, field.PosT)
	scratch := make([]uint32, field.PosT)
	for i := 0; i < field.PosT; i++ {
		state[i] = uint32(i + 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(scratch, state)
		scratchArr := (*[field.PosT]uint32)(scratch)
		for row := 0; row < field.PosT; row++ {
			var acc uint64
			invSlice := (*[field.PosT]uint32)(PosInv[row : row+field.PosT])
			for j := 0; j < 35; j += 7 {
				t0 := uint64(invSlice[j]) * uint64(scratchArr[j])
				t1 := uint64(invSlice[j+1]) * uint64(scratchArr[j+1])
				t2 := uint64(invSlice[j+2]) * uint64(scratchArr[j+2])
				t3 := uint64(invSlice[j+3]) * uint64(scratchArr[j+3])
				t4 := uint64(invSlice[j+4]) * uint64(scratchArr[j+4])
				t5 := uint64(invSlice[j+5]) * uint64(scratchArr[j+5])
				t6 := uint64(invSlice[j+6]) * uint64(scratchArr[j+6])
				acc += t0 + t1 + t2 + t3 + t4 + t5 + t6
			}
			state[row] = uint32(acc % field.Q)
		}
	}
}

// Benchmark MDS with 2-row parallel
func BenchmarkMDS2Row(b *testing.B) {
	state := make([]uint32, field.PosT)
	scratch := make([]uint32, field.PosT)
	for i := 0; i < field.PosT; i++ {
		state[i] = uint32(i + 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(scratch, state)
		scratchArr := (*[field.PosT]uint32)(scratch)
		// Process 2 rows at a time
		for row := 0; row < 34; row += 2 {
			var acc0, acc1 uint64
			invSlice0 := (*[field.PosT]uint32)(PosInv[row : row+field.PosT])
			invSlice1 := (*[field.PosT]uint32)(PosInv[row+1 : row+1+field.PosT])
			for j := 0; j < 35; j += 5 {
				s0 := uint64(scratchArr[j])
				s1 := uint64(scratchArr[j+1])
				s2 := uint64(scratchArr[j+2])
				s3 := uint64(scratchArr[j+3])
				s4 := uint64(scratchArr[j+4])
				acc0 += uint64(invSlice0[j])*s0 + uint64(invSlice0[j+1])*s1 +
					uint64(invSlice0[j+2])*s2 + uint64(invSlice0[j+3])*s3 +
					uint64(invSlice0[j+4])*s4
				acc1 += uint64(invSlice1[j])*s0 + uint64(invSlice1[j+1])*s1 +
					uint64(invSlice1[j+2])*s2 + uint64(invSlice1[j+3])*s3 +
					uint64(invSlice1[j+4])*s4
			}
			state[row] = uint32(acc0 % field.Q)
			state[row+1] = uint32(acc1 % field.Q)
		}
		// Handle last row (35 is odd)
		var acc uint64
		invSlice := (*[field.PosT]uint32)(PosInv[34 : 34+field.PosT])
		for j := 0; j < 35; j += 5 {
			t0 := uint64(invSlice[j]) * uint64(scratchArr[j])
			t1 := uint64(invSlice[j+1]) * uint64(scratchArr[j+1])
			t2 := uint64(invSlice[j+2]) * uint64(scratchArr[j+2])
			t3 := uint64(invSlice[j+3]) * uint64(scratchArr[j+3])
			t4 := uint64(invSlice[j+4]) * uint64(scratchArr[j+4])
			acc += t0 + t1 + t2 + t3 + t4
		}
		state[34] = uint32(acc % field.Q)
	}
}

// Benchmark full poseidon round
func BenchmarkPoseidonRound(b *testing.B) {
	state := make([]uint32, field.PosT)
	scratch := make([]uint32, 3*field.PosT)
	for i := 0; i < field.PosT; i++ {
		state[i] = uint32(i + 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		poseidonRound(state, scratch, 0)
	}
}

// Benchmark full poseidon perm
func BenchmarkPoseidonPerm(b *testing.B) {
	state := make([]uint32, field.PosT)
	for i := 0; i < field.PosT; i++ {
		state[i] = uint32(i + 1)
	}
	orig := make([]uint32, field.PosT)
	copy(orig, state)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(state, orig)
		PoseidonPerm(state)
	}
}
