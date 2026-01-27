package sampling

import (
	"testing"
	"zkdilithium-signer/pkg/field"
	"zkdilithium-signer/pkg/hash"
)

// Test SampleUniform first 16 values from Python
func TestSampleUniformFirst16(t *testing.T) {
	stream := hash.XOF128(make([]byte, 32), 0)
	p := SampleUniform(stream)

	expected := []uint32{
		5889865, 3971968, 4850004, 6999211, 2967789, 1694039, 636417, 4598392,
		7167687, 1092265, 3028014, 5070791, 5596185, 3786936, 6256060, 5896089,
	}
	for i, want := range expected {
		if p[i] != want {
			t.Errorf("SampleUniform[%d] = %d, want %d", i, p[i], want)
		}
	}
}

// Test SampleUniform last 16 values from Python
func TestSampleUniformLast16(t *testing.T) {
	stream := hash.XOF128(make([]byte, 32), 0)
	p := SampleUniform(stream)

	expected := []uint32{
		1649304, 4661824, 3620918, 6844818, 2645999, 3739555, 3888682, 4274156,
		6815638, 3786571, 4509883, 4371144, 2001635, 1862166, 3110494, 3082926,
	}
	for i, want := range expected {
		if p[field.N-16+i] != want {
			t.Errorf("SampleUniform[%d] = %d, want %d", field.N-16+i, p[field.N-16+i], want)
		}
	}
}

// Test SampleLeqEta first 16 values from Python
func TestSampleLeqEtaFirst16(t *testing.T) {
	stream := hash.XOF256(make([]byte, 64), 0)
	p := SampleLeqEta(stream)

	// Note: negative values are stored as Q-|value|
	expected := []uint32{0, 7340031, 7340032, 7340032, 0, 7340032, 0, 2,
		0, 7340032, 2, 7340032, 2, 1, 7340032, 2}
	for i, want := range expected {
		if p[i] != want {
			t.Errorf("SampleLeqEta[%d] = %d, want %d", i, p[i], want)
		}
	}
}

// Test SampleLeqEta produces valid coefficients
func TestSampleLeqEtaBounds(t *testing.T) {
	stream := hash.XOF256(make([]byte, 64), 0)
	p := SampleLeqEta(stream)

	for i, c := range p {
		// c should be in {0, 1, 2, Q-1, Q-2}
		if c > field.Eta && c < field.Q-field.Eta {
			t.Errorf("SampleLeqEta[%d] = %d out of bounds", i, c)
		}
	}
}

// Test SampleMatrix dimensions
func TestSampleMatrixDimensions(t *testing.T) {
	rho := make([]byte, 32)
	A := SampleMatrix(rho)

	if len(A) != field.K {
		t.Errorf("SampleMatrix rows = %d, want %d", len(A), field.K)
	}
	for i := 0; i < field.K; i++ {
		if len(A[i]) != field.L {
			t.Errorf("SampleMatrix cols = %d, want %d", len(A[i]), field.L)
		}
	}
}

// Test SampleSecret dimensions
func TestSampleSecretDimensions(t *testing.T) {
	rho := make([]byte, 64)
	s1, s2 := SampleSecret(rho)

	if len(s1) != field.L {
		t.Errorf("s1 length = %d, want %d", len(s1), field.L)
	}
	if len(s2) != field.K {
		t.Errorf("s2 length = %d, want %d", len(s2), field.K)
	}
}

// Test SampleY norm bound
func TestSampleYNormBound(t *testing.T) {
	rho := make([]byte, 64)
	y := SampleY(rho, 0)

	for i := 0; i < field.L; i++ {
		var p [field.N]uint32 = y[i]
		// Check norm < Gamma1
		var maxNorm uint32
		for _, c := range p {
			var absC uint32
			if c > field.Q/2 {
				absC = field.Q - c
			} else {
				absC = c
			}
			if absC > maxNorm {
				maxNorm = absC
			}
		}
		if maxNorm >= field.Gamma1 {
			t.Errorf("SampleY[%d] norm = %d >= Gamma1", i, maxNorm)
		}
	}
}

// Test SampleInBall produces expected positions and values
func TestSampleInBall(t *testing.T) {
	h := hash.NewPoseidon([]uint32{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	c := SampleInBall(h)

	if c == nil {
		t.Fatal("SampleInBall returned nil")
	}

	// Count non-zero coefficients
	var nonzero []struct {
		pos int
		val uint32
	}
	for i := 0; i < field.N; i++ {
		if c[i] != 0 {
			nonzero = append(nonzero, struct {
				pos int
				val uint32
			}{i, c[i]})
		}
	}

	if len(nonzero) != field.Tau {
		t.Errorf("SampleInBall nonzero count = %d, want %d", len(nonzero), field.Tau)
	}

	// Check first 10 positions and values from Python
	expectedPositions := []int{11, 17, 24, 42, 50, 51, 57, 58, 61, 70}
	expectedValues := []uint32{1, 1, 1, 7340032, 7340032, 7340032, 1, 7340032, 7340032, 7340032}

	for i := 0; i < 10 && i < len(nonzero); i++ {
		if nonzero[i].pos != expectedPositions[i] {
			t.Errorf("SampleInBall nonzero[%d].pos = %d, want %d", i, nonzero[i].pos, expectedPositions[i])
		}
		if nonzero[i].val != expectedValues[i] {
			t.Errorf("SampleInBall nonzero[%d].val = %d, want %d", i, nonzero[i].val, expectedValues[i])
		}
	}
}

// Test all non-zero coefficients are ±1
func TestSampleInBallCoeffs(t *testing.T) {
	h := hash.NewPoseidon([]uint32{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	c := SampleInBall(h)

	if c == nil {
		t.Fatal("SampleInBall returned nil")
	}

	for i, coeff := range c {
		if coeff != 0 && coeff != 1 && coeff != field.Q-1 {
			t.Errorf("SampleInBall[%d] = %d, want 0, 1, or Q-1", i, coeff)
		}
	}
}
