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

// Test round constants first 16 match Python
func TestPosRCsFirst16(t *testing.T) {
	expected := []uint32{
		662000, 7104925, 2304656, 2330809, 452951, 1722141, 5334010, 7087604,
		5110463, 6023804, 3061965, 6087945, 3740272, 284272, 4421217, 559188,
	}
	for i, want := range expected {
		if PosRCs[i] != want {
			t.Errorf("PosRCs[%d] = %d, want %d", i, PosRCs[i], want)
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
		if state[i] != want {
			t.Errorf("PoseidonPerm[%d] = %d, want %d", i, state[i], want)
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
