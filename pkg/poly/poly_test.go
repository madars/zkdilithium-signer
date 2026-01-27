package poly

import (
	"testing"
	"zkdilithium-signer/pkg/field"
)

// Test schoolbook multiplication result first 16 (from Python)
func TestSchoolbookMulResultFirst16(t *testing.T) {
	var a, b Poly
	for i := 0; i < field.N; i++ {
		a[i] = uint32(i)
		b[i] = uint32(i + 256)
	}

	_, r := SchoolbookMul(&a, &b)

	expected := []uint32{
		3528066, 3496194, 3465092, 3434762, 3405206, 3376426, 3348424, 3321202,
		3294762, 3269106, 3244236, 3220154, 3196862, 3174362, 3152656, 3131746,
	}
	for i, want := range expected {
		if r[i] != want {
			t.Errorf("SchoolbookMul result[%d] = %d, want %d", i, r[i], want)
		}
	}
}

// Test schoolbook multiplication result last 16 (from Python)
func TestSchoolbookMulResultLast16(t *testing.T) {
	var a, b Poly
	for i := 0; i < field.N; i++ {
		a[i] = uint32(i)
		b[i] = uint32(i + 256)
	}

	_, r := SchoolbookMul(&a, &b)

	expected := []uint32{
		492847, 703135, 914673, 1127463, 1341507, 1556807, 1773365, 1991183,
		2210263, 2430607, 2652217, 2875095, 3099243, 3324663, 3551357, 3779327,
	}
	for i, want := range expected {
		if r[field.N-16+i] != want {
			t.Errorf("SchoolbookMul result[%d] = %d, want %d", field.N-16+i, r[field.N-16+i], want)
		}
	}
}

// Test schoolbook multiplication quotient first 16 (from Python)
func TestSchoolbookMulQuotientFirst16(t *testing.T) {
	var a, b Poly
	for i := 0; i < field.N; i++ {
		a[i] = uint32(i)
		b[i] = uint32(i + 256)
	}

	q, _ := SchoolbookMul(&a, &b)

	expected := []uint32{
		3811967, 3844095, 3875710, 3906811, 3937397, 3967467, 3997020, 4026055,
		4054571, 4082567, 4110042, 4136995, 4163425, 4189331, 4214712, 4239567,
	}
	for i, want := range expected {
		if q[i] != want {
			t.Errorf("SchoolbookMul quotient[%d] = %d, want %d", i, q[i], want)
		}
	}
}

// Test NTT multiplication matches schoolbook (catches incorrect implementations)
func TestNTTMulMatchesSchoolbook(t *testing.T) {
	var a, b Poly
	for i := 0; i < field.N; i++ {
		a[i] = uint32(i)
		b[i] = uint32(i + 256)
	}

	// Schoolbook multiplication
	_, rSchool := SchoolbookMul(&a, &b)

	// NTT multiplication
	var aNTT, bNTT, rNTT Poly
	Copy(&aNTT, &a)
	Copy(&bNTT, &b)
	aNTT.NTT()
	bNTT.NTT()
	MulNTT(&aNTT, &bNTT, &rNTT)
	rNTT.InvNTT()

	if !Equal(&rNTT, &rSchool) {
		t.Error("NTT multiplication does not match schoolbook")
		// Find first difference
		for i := 0; i < field.N; i++ {
			if rNTT[i] != rSchool[i] {
				t.Errorf("First diff at [%d]: NTT=%d, schoolbook=%d", i, rNTT[i], rSchool[i])
				break
			}
		}
	}
}

// Test multiply by 1 returns original
func TestSchoolbookMulByOne(t *testing.T) {
	var a, one Poly
	for i := 0; i < field.N; i++ {
		a[i] = uint32(i)
	}
	one[0] = 1 // one = 1 + 0*x + 0*x^2 + ...

	q, r := SchoolbookMul(&a, &one)

	// r should equal a
	if !Equal(&r, &a) {
		t.Error("Multiplying by 1 does not return original")
	}

	// q should be all zeros
	for i := 0; i < field.N; i++ {
		if q[i] != 0 {
			t.Errorf("Quotient[%d] = %d, want 0", i, q[i])
		}
	}
}

// Test polynomial addition
func TestPolyAdd(t *testing.T) {
	var a, b, result Poly
	for i := 0; i < field.N; i++ {
		a[i] = 1
		b[i] = 2
	}

	Add(&a, &b, &result)

	for i := 0; i < field.N; i++ {
		if result[i] != 3 {
			t.Errorf("Add result[%d] = %d, want 3", i, result[i])
		}
	}
}

// Test polynomial subtraction
func TestPolySub(t *testing.T) {
	var a, b, result Poly
	for i := 0; i < field.N; i++ {
		a[i] = 5
		b[i] = 2
	}

	Sub(&a, &b, &result)

	for i := 0; i < field.N; i++ {
		if result[i] != 3 {
			t.Errorf("Sub result[%d] = %d, want 3", i, result[i])
		}
	}
}

// Test polynomial norm
func TestPolyNorm(t *testing.T) {
	var p Poly
	p[0] = 5
	p[1] = 10
	p[2] = 3

	if p.Norm() != 10 {
		t.Errorf("Norm = %d, want 10", p.Norm())
	}

	// Test with negative (stored as Q-x)
	p[3] = field.Q - 100 // represents -100
	if p.Norm() != 100 {
		t.Errorf("Norm with negative = %d, want 100", p.Norm())
	}
}

// Test polynomial decompose
func TestPolyDecompose(t *testing.T) {
	var p Poly
	p[0] = 0
	p[1] = 1
	p[2] = 131072 // 2*Gamma2
	p[3] = field.Q - 1

	p0, p1 := p.Decompose()

	// Check specific values from Python
	r0, r1 := field.Decompose(p[0])
	if p0[0] != field.Mod(int64(r0)) || p1[0] != r1 {
		t.Errorf("Decompose[0] mismatch")
	}

	r0, r1 = field.Decompose(p[2])
	if p0[2] != field.Mod(int64(r0)) || p1[2] != r1 {
		t.Errorf("Decompose[2]: got (%d, %d), want (%d, %d)", p0[2], p1[2], field.Mod(int64(r0)), r1)
	}
}
