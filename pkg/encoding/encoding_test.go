package encoding

import (
	"bytes"
	"encoding/hex"
	"testing"
	"zkdilithium-signer/pkg/field"
)

// Test BytesToFes with known values from Python
func TestBytesToFes(t *testing.T) {
	tests := []struct {
		input []byte
		want  []uint32
	}{
		{[]byte{0, 0}, []uint32{258}},
		{[]byte{5}, []uint32{6}},
		{[]byte{0xFF, 0xFF}, []uint32{66048}},
		{[]byte("hello"), []uint32{26319, 28122, 112}},
	}
	for _, tc := range tests {
		got := BytesToFes(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("BytesToFes(%v): len=%d, want %d", tc.input, len(got), len(tc.want))
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("BytesToFes(%v)[%d] = %d, want %d", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}

// Test PackFes with known values from Python
func TestPackFes(t *testing.T) {
	fes := []uint32{0, 1, 100, 1000, 7340032, 3670016}
	expected, _ := hex.DecodeString("000000010000640000e80300000070000038")
	got := PackFes(fes)
	if !bytes.Equal(got, expected) {
		t.Errorf("PackFes: got %x, want %x", got, expected)
	}
}

// Test PackFes/UnpackFes roundtrip
func TestPackFesRoundtrip(t *testing.T) {
	fes := []uint32{0, 1, 100, 1000, field.Q - 1, field.Q / 2}
	unpacked := UnpackFes(PackFes(fes))
	for i := range fes {
		if unpacked[i] != fes[i] {
			t.Errorf("Roundtrip[%d]: got %d, want %d", i, unpacked[i], fes[i])
		}
	}
}

// Test PackPolyLeqEta/UnpackPolyLeqEta roundtrip
func TestPackPolyLeqEtaRoundtrip(t *testing.T) {
	var p [field.N]uint32
	// Fill with values in [-Eta, Eta] mod Q
	for i := 0; i < field.N; i++ {
		switch i % 5 {
		case 0:
			p[i] = 0
		case 1:
			p[i] = 1
		case 2:
			p[i] = 2
		case 3:
			p[i] = field.Q - 1 // -1
		case 4:
			p[i] = field.Q - 2 // -2
		}
	}

	packed := PackPolyLeqEta(&p)
	unpacked := UnpackPolyLeqEta(packed)

	for i := 0; i < field.N; i++ {
		if unpacked[i] != p[i] {
			t.Errorf("LeqEta roundtrip[%d]: got %d, want %d", i, unpacked[i], p[i])
		}
	}
}

// Test PackPolyLeGamma1/UnpackPolyLeGamma1 roundtrip
func TestPackPolyLeGamma1Roundtrip(t *testing.T) {
	var p [field.N]uint32
	// Fill with values in [-Gamma1+1, Gamma1] mod Q
	for i := 0; i < field.N; i++ {
		switch i % 4 {
		case 0:
			p[i] = 0
		case 1:
			p[i] = field.Gamma1
		case 2:
			p[i] = field.Mod(-int64(field.Gamma1) + 1) // -Gamma1+1
		case 3:
			p[i] = uint32(i % field.Gamma1)
		}
	}

	packed := PackPolyLeGamma1(&p)
	unpacked := UnpackPolyLeGamma1(packed)

	for i := 0; i < field.N; i++ {
		if unpacked[i] != p[i] {
			t.Errorf("LeGamma1 roundtrip[%d]: got %d, want %d", i, unpacked[i], p[i])
		}
	}
}

// Test packed sizes are correct
func TestPackedSizes(t *testing.T) {
	var p [field.N]uint32

	if len(PackPoly(&p)) != 256*3 {
		t.Errorf("PackPoly size: %d, want %d", len(PackPoly(&p)), 256*3)
	}

	if len(PackPolyLeqEta(&p)) != 96 {
		t.Errorf("PackPolyLeqEta size: %d, want 96", len(PackPolyLeqEta(&p)))
	}

	if len(PackPolyLeGamma1(&p)) != field.PolyLeGamma1Size {
		t.Errorf("PackPolyLeGamma1 size: %d, want %d", len(PackPolyLeGamma1(&p)), field.PolyLeGamma1Size)
	}
}
