package field

import "testing"

func TestBatchInvTreeCondPlainMatchesBatchInv(t *testing.T) {
	for iter := 0; iter < 2000; iter++ {
		var xs0 [PosT]uint32
		var xs1 [PosT]uint32
		for i := 0; i < PosT; i++ {
			v := uint32((iter*977 + i*131 + 1) % int(Q))
			if v == 0 {
				v = 1
			}
			xs0[i] = v
			xs1[i] = v
		}

		BatchInv(xs0[:])
		scratch := make([]uint32, 128)
		BatchInvTreeCondPlain(xs1[:], scratch)

		for i := 0; i < PosT; i++ {
			if xs1[i] != xs0[i] {
				t.Fatalf("iter=%d idx=%d got=%d want=%d", iter, i, xs1[i], xs0[i])
			}
		}
	}
}

func TestBatchInvTreeCondPlainWithZeros(t *testing.T) {
	xs0 := []uint32{1, 0, 2, 0, 3, 4, 0}
	xs1 := make([]uint32, len(xs0))
	copy(xs1, xs0)

	BatchInv(xs0)
	scratch := make([]uint32, 64)
	BatchInvTreeCondPlain(xs1, scratch)

	for i := range xs0 {
		if xs1[i] != xs0[i] {
			t.Fatalf("idx=%d got=%d want=%d", i, xs1[i], xs0[i])
		}
	}
}
