package field

import "testing"

func TestBatchInvMontTreeNoZeroILP4_35LazyPlainMatchesOrig(t *testing.T) {
	for iter := 0; iter < 5000; iter++ {
		var xsOrig [PosT]uint32
		var xsNew [PosT]uint32
		for i := 0; i < PosT; i++ {
			v := uint32((iter*977 + i*131 + 1) % int(Q))
			if v == 0 {
				v = 1
			}
			m := ToMont(v)
			xsOrig[i] = m
			xsNew[i] = m
		}

		scratchOrig := make([]uint32, 128)
		scratchNew := make([]uint32, 128)
		batchInvMontTreeNoZeroILP4_35(xsOrig[:], scratchOrig)
		batchInvMontTreeNoZeroILP4_35LazyPlain(xsNew[:], scratchNew)

		for i := 0; i < PosT; i++ {
			if xsOrig[i] != xsNew[i] {
				t.Fatalf("iter=%d idx=%d got=%d want=%d", iter, i, xsNew[i], xsOrig[i])
			}
		}
	}
}

func TestBatchInvMontTreeNoZeroILP4_35LazyPlainProperty(t *testing.T) {
	for iter := 0; iter < 2000; iter++ {
		var xs [PosT]uint32
		var orig [PosT]uint32
		for i := 0; i < PosT; i++ {
			v := uint32((iter*353 + i*19 + 1) % int(Q))
			if v == 0 {
				v = 1
			}
			m := ToMont(v)
			xs[i] = m
			orig[i] = m
		}

		scratch := make([]uint32, 128)
		batchInvMontTreeNoZeroILP4_35LazyPlain(xs[:], scratch)

		oneM := ToMont(1)
		for i := 0; i < PosT; i++ {
			if got := MulMont(orig[i], xs[i]); got != oneM {
				t.Fatalf("iter=%d idx=%d product=%d want=%d", iter, i, got, oneM)
			}
		}
	}
}

func BenchmarkBatchInvTreeNoZeroILP4_35MontOrigDirect(b *testing.B) {
	xs := make([]uint32, PosT)
	scratch := make([]uint32, 128)
	for i := 0; i < PosT; i++ {
		xs[i] = ToMont(uint32(i + 1))
	}
	orig := make([]uint32, PosT)
	copy(orig, xs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(xs, orig)
		batchInvMontTreeNoZeroILP4_35(xs, scratch)
	}
}

func BenchmarkBatchInvTreeNoZeroILP4_35LazyPlainDirect(b *testing.B) {
	xs := make([]uint32, PosT)
	scratch := make([]uint32, 128)
	for i := 0; i < PosT; i++ {
		xs[i] = ToMont(uint32(i + 1))
	}
	orig := make([]uint32, PosT)
	copy(orig, xs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(xs, orig)
		batchInvMontTreeNoZeroILP4_35LazyPlain(xs, scratch)
	}
}
