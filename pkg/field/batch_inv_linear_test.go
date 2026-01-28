package field

import "testing"

func TestBatchInvMontLinear(t *testing.T) {
	n := PosT // 35

	xs := make([]uint32, n)
	xsRef := make([]uint32, n)
	for i := range xs {
		xs[i] = ToMont(uint32(i + 1))
		xsRef[i] = xs[i]
	}

	scratch := make([]uint32, n)
	scratchRef := make([]uint32, n)

	BatchInvMontLinear(xs, scratch)
	BatchInvMont(xsRef, scratchRef)

	for i := range xs {
		if xs[i] != xsRef[i] {
			t.Errorf("index %d: got %d, want %d", i, xs[i], xsRef[i])
		}
	}
}

func TestBatchInvMontLinearWithZeros(t *testing.T) {
	xs := []uint32{ToMont(1), 0, ToMont(3), 0, ToMont(5)}
	xsRef := make([]uint32, len(xs))
	copy(xsRef, xs)

	scratch := make([]uint32, len(xs))
	scratchRef := make([]uint32, len(xs))

	BatchInvMontLinear(xs, scratch)
	BatchInvMont(xsRef, scratchRef)

	for i := range xs {
		if xs[i] != xsRef[i] {
			t.Errorf("index %d: got %d, want %d", i, xs[i], xsRef[i])
		}
	}
}

func TestBatchInvMontLinearOddN(t *testing.T) {
	// Test odd-sized array
	for n := 1; n <= 10; n++ {
		xs := make([]uint32, n)
		xsRef := make([]uint32, n)
		for i := range xs {
			xs[i] = ToMont(uint32(i + 1))
			xsRef[i] = xs[i]
		}

		scratch := make([]uint32, n)
		scratchRef := make([]uint32, n)

		BatchInvMontLinear(xs, scratch)
		BatchInvMont(xsRef, scratchRef)

		for i := range xs {
			if xs[i] != xsRef[i] {
				t.Errorf("n=%d index %d: got %d, want %d", n, i, xs[i], xsRef[i])
			}
		}
	}
}

func TestBatchInvMontLinearCorrectness(t *testing.T) {
	n := 35
	xs := make([]uint32, n)
	original := make([]uint32, n)

	for i := range xs {
		xs[i] = ToMont(uint32(i + 1))
		original[i] = xs[i]
	}

	scratch := make([]uint32, n)
	BatchInvMontLinear(xs, scratch)

	oneM := ToMont(1)
	for i := range xs {
		product := MulMont(original[i], xs[i])
		if product != oneM {
			t.Errorf("x * x^(-1) != 1 at index %d: got %d, want %d", i, product, oneM)
		}
	}
}

func BenchmarkBatchInvMontLinear(b *testing.B) {
	xs := make([]uint32, PosT)
	scratch := make([]uint32, PosT)
	for i := range xs {
		xs[i] = ToMont(uint32(i + 1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range xs {
			xs[j] = ToMont(uint32(j + 1))
		}
		BatchInvMontLinear(xs, scratch)
	}
}

func BenchmarkBatchInvMontOriginal2(b *testing.B) {
	xs := make([]uint32, PosT)
	scratch := make([]uint32, PosT)
	for i := range xs {
		xs[i] = ToMont(uint32(i + 1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range xs {
			xs[j] = ToMont(uint32(j + 1))
		}
		BatchInvMont(xs, scratch)
	}
}
