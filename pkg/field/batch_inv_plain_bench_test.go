package field

import "testing"

func batchInvTreeNoZeroILP4_35Plain(xs []uint32, scratch []uint32) {
	x := (*[PosT]uint32)(xs)    // 35
	s := (*[38]uint32)(scratch) // 18+9+5+3+2+1

	// ============ UP-SWEEP ============
	s[0] = Mul(x[0], x[1])
	s[1] = Mul(x[2], x[3])
	s[2] = Mul(x[4], x[5])
	s[3] = Mul(x[6], x[7])
	s[4] = Mul(x[8], x[9])
	s[5] = Mul(x[10], x[11])
	s[6] = Mul(x[12], x[13])
	s[7] = Mul(x[14], x[15])
	s[8] = Mul(x[16], x[17])
	s[9] = Mul(x[18], x[19])
	s[10] = Mul(x[20], x[21])
	s[11] = Mul(x[22], x[23])
	s[12] = Mul(x[24], x[25])
	s[13] = Mul(x[26], x[27])
	s[14] = Mul(x[28], x[29])
	s[15] = Mul(x[30], x[31])
	s[16] = Mul(x[32], x[33])
	s[17] = x[34]

	s[18] = Mul(s[0], s[1])
	s[19] = Mul(s[2], s[3])
	s[20] = Mul(s[4], s[5])
	s[21] = Mul(s[6], s[7])
	s[22] = Mul(s[8], s[9])
	s[23] = Mul(s[10], s[11])
	s[24] = Mul(s[12], s[13])
	s[25] = Mul(s[14], s[15])
	s[26] = Mul(s[16], s[17])

	s[27] = Mul(s[18], s[19])
	s[28] = Mul(s[20], s[21])
	s[29] = Mul(s[22], s[23])
	s[30] = Mul(s[24], s[25])
	s[31] = s[26]

	s[32] = Mul(s[27], s[28])
	s[33] = Mul(s[29], s[30])
	s[34] = s[31]

	s[35] = Mul(s[32], s[33])
	s[36] = s[34]

	// ============ INVERT ROOT ============
	s[37] = Inv(Mul(s[35], s[36]))

	// ============ DOWN-SWEEP ============
	parentInv := s[37]
	leftVal := s[35]
	rightVal := s[36]
	s[35] = Mul(parentInv, rightVal)
	s[36] = Mul(parentInv, leftVal)

	parentInv = s[35]
	leftVal = s[32]
	rightVal = s[33]
	s[32] = Mul(parentInv, rightVal)
	s[33] = Mul(parentInv, leftVal)
	s[34] = s[36]

	parentInv = s[32]
	leftVal = s[27]
	rightVal = s[28]
	s[27] = Mul(parentInv, rightVal)
	s[28] = Mul(parentInv, leftVal)

	parentInv = s[33]
	leftVal = s[29]
	rightVal = s[30]
	s[29] = Mul(parentInv, rightVal)
	s[30] = Mul(parentInv, leftVal)
	s[31] = s[34]

	parentInv = s[27]
	leftVal = s[18]
	rightVal = s[19]
	s[18] = Mul(parentInv, rightVal)
	s[19] = Mul(parentInv, leftVal)

	parentInv = s[28]
	leftVal = s[20]
	rightVal = s[21]
	s[20] = Mul(parentInv, rightVal)
	s[21] = Mul(parentInv, leftVal)

	parentInv = s[29]
	leftVal = s[22]
	rightVal = s[23]
	s[22] = Mul(parentInv, rightVal)
	s[23] = Mul(parentInv, leftVal)

	parentInv = s[30]
	leftVal = s[24]
	rightVal = s[25]
	s[24] = Mul(parentInv, rightVal)
	s[25] = Mul(parentInv, leftVal)
	s[26] = s[31]

	parentInv = s[18]
	leftVal = s[0]
	rightVal = s[1]
	s[0] = Mul(parentInv, rightVal)
	s[1] = Mul(parentInv, leftVal)

	parentInv = s[19]
	leftVal = s[2]
	rightVal = s[3]
	s[2] = Mul(parentInv, rightVal)
	s[3] = Mul(parentInv, leftVal)

	parentInv = s[20]
	leftVal = s[4]
	rightVal = s[5]
	s[4] = Mul(parentInv, rightVal)
	s[5] = Mul(parentInv, leftVal)

	parentInv = s[21]
	leftVal = s[6]
	rightVal = s[7]
	s[6] = Mul(parentInv, rightVal)
	s[7] = Mul(parentInv, leftVal)

	parentInv = s[22]
	leftVal = s[8]
	rightVal = s[9]
	s[8] = Mul(parentInv, rightVal)
	s[9] = Mul(parentInv, leftVal)

	parentInv = s[23]
	leftVal = s[10]
	rightVal = s[11]
	s[10] = Mul(parentInv, rightVal)
	s[11] = Mul(parentInv, leftVal)

	parentInv = s[24]
	leftVal = s[12]
	rightVal = s[13]
	s[12] = Mul(parentInv, rightVal)
	s[13] = Mul(parentInv, leftVal)

	parentInv = s[25]
	leftVal = s[14]
	rightVal = s[15]
	s[14] = Mul(parentInv, rightVal)
	s[15] = Mul(parentInv, leftVal)

	parentInv = s[26]
	leftVal = s[16]
	rightVal = s[17]
	s[16] = Mul(parentInv, rightVal)
	s[17] = Mul(parentInv, leftVal)

	// Final layer writes strictly reduced values, eliminating trailing reduce pass.
	parentInv = s[0]
	leftVal = x[0]
	rightVal = x[1]
	x[0] = Mul(parentInv, rightVal)
	x[1] = Mul(parentInv, leftVal)

	parentInv = s[1]
	leftVal = x[2]
	rightVal = x[3]
	x[2] = Mul(parentInv, rightVal)
	x[3] = Mul(parentInv, leftVal)

	parentInv = s[2]
	leftVal = x[4]
	rightVal = x[5]
	x[4] = Mul(parentInv, rightVal)
	x[5] = Mul(parentInv, leftVal)

	parentInv = s[3]
	leftVal = x[6]
	rightVal = x[7]
	x[6] = Mul(parentInv, rightVal)
	x[7] = Mul(parentInv, leftVal)

	parentInv = s[4]
	leftVal = x[8]
	rightVal = x[9]
	x[8] = Mul(parentInv, rightVal)
	x[9] = Mul(parentInv, leftVal)

	parentInv = s[5]
	leftVal = x[10]
	rightVal = x[11]
	x[10] = Mul(parentInv, rightVal)
	x[11] = Mul(parentInv, leftVal)

	parentInv = s[6]
	leftVal = x[12]
	rightVal = x[13]
	x[12] = Mul(parentInv, rightVal)
	x[13] = Mul(parentInv, leftVal)

	parentInv = s[7]
	leftVal = x[14]
	rightVal = x[15]
	x[14] = Mul(parentInv, rightVal)
	x[15] = Mul(parentInv, leftVal)

	parentInv = s[8]
	leftVal = x[16]
	rightVal = x[17]
	x[16] = Mul(parentInv, rightVal)
	x[17] = Mul(parentInv, leftVal)

	parentInv = s[9]
	leftVal = x[18]
	rightVal = x[19]
	x[18] = Mul(parentInv, rightVal)
	x[19] = Mul(parentInv, leftVal)

	parentInv = s[10]
	leftVal = x[20]
	rightVal = x[21]
	x[20] = Mul(parentInv, rightVal)
	x[21] = Mul(parentInv, leftVal)

	parentInv = s[11]
	leftVal = x[22]
	rightVal = x[23]
	x[22] = Mul(parentInv, rightVal)
	x[23] = Mul(parentInv, leftVal)

	parentInv = s[12]
	leftVal = x[24]
	rightVal = x[25]
	x[24] = Mul(parentInv, rightVal)
	x[25] = Mul(parentInv, leftVal)

	parentInv = s[13]
	leftVal = x[26]
	rightVal = x[27]
	x[26] = Mul(parentInv, rightVal)
	x[27] = Mul(parentInv, leftVal)

	parentInv = s[14]
	leftVal = x[28]
	rightVal = x[29]
	x[28] = Mul(parentInv, rightVal)
	x[29] = Mul(parentInv, leftVal)

	parentInv = s[15]
	leftVal = x[30]
	rightVal = x[31]
	x[30] = Mul(parentInv, rightVal)
	x[31] = Mul(parentInv, leftVal)

	parentInv = s[16]
	leftVal = x[32]
	rightVal = x[33]
	x[32] = Mul(parentInv, rightVal)
	x[33] = Mul(parentInv, leftVal)

	x[34] = s[17]
}

func batchInvTreeNoZeroILP4_35PlainLazy(xs []uint32, scratch []uint32) {
	x := (*[PosT]uint32)(xs)    // 35
	s := (*[38]uint32)(scratch) // 18+9+5+3+2+1

	// ============ UP-SWEEP ============
	s[0], s[1] = mulPlainLazy2(x[0], x[1], x[2], x[3])
	s[2], s[3] = mulPlainLazy2(x[4], x[5], x[6], x[7])
	s[4], s[5] = mulPlainLazy2(x[8], x[9], x[10], x[11])
	s[6], s[7] = mulPlainLazy2(x[12], x[13], x[14], x[15])
	s[8], s[9] = mulPlainLazy2(x[16], x[17], x[18], x[19])
	s[10], s[11] = mulPlainLazy2(x[20], x[21], x[22], x[23])
	s[12], s[13] = mulPlainLazy2(x[24], x[25], x[26], x[27])
	s[14], s[15] = mulPlainLazy2(x[28], x[29], x[30], x[31])
	s[16] = mulPlainLazy(x[32], x[33])
	s[17] = x[34]

	s[18], s[19] = mulPlainLazy2(s[0], s[1], s[2], s[3])
	s[20], s[21] = mulPlainLazy2(s[4], s[5], s[6], s[7])
	s[22], s[23] = mulPlainLazy2(s[8], s[9], s[10], s[11])
	s[24], s[25] = mulPlainLazy2(s[12], s[13], s[14], s[15])
	s[26] = mulPlainLazy(s[16], s[17])

	s[27], s[28] = mulPlainLazy2(s[18], s[19], s[20], s[21])
	s[29], s[30] = mulPlainLazy2(s[22], s[23], s[24], s[25])
	s[31] = s[26]

	s[32] = mulPlainLazy(s[27], s[28])
	s[33] = mulPlainLazy(s[29], s[30])
	s[34] = s[31]

	s[35] = mulPlainLazy(s[32], s[33])
	s[36] = s[34]

	// ============ INVERT ROOT ============
	rootProd := mulPlainLazy(s[35], s[36])
	s[37] = invPlainLazy(rootProd)

	// ============ DOWN-SWEEP ============
	parentInv := s[37]
	leftVal := s[35]
	rightVal := s[36]
	s[35], s[36] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[35]
	leftVal = s[32]
	rightVal = s[33]
	s[32], s[33] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)
	s[34] = s[36]

	parentInv = s[32]
	leftVal = s[27]
	rightVal = s[28]
	s[27], s[28] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[33]
	leftVal = s[29]
	rightVal = s[30]
	s[29], s[30] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)
	s[31] = s[34]

	parentInv = s[27]
	leftVal = s[18]
	rightVal = s[19]
	s[18], s[19] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[28]
	leftVal = s[20]
	rightVal = s[21]
	s[20], s[21] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[29]
	leftVal = s[22]
	rightVal = s[23]
	s[22], s[23] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[30]
	leftVal = s[24]
	rightVal = s[25]
	s[24], s[25] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)
	s[26] = s[31]

	parentInv = s[18]
	leftVal = s[0]
	rightVal = s[1]
	s[0], s[1] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[19]
	leftVal = s[2]
	rightVal = s[3]
	s[2], s[3] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[20]
	leftVal = s[4]
	rightVal = s[5]
	s[4], s[5] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[21]
	leftVal = s[6]
	rightVal = s[7]
	s[6], s[7] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[22]
	leftVal = s[8]
	rightVal = s[9]
	s[8], s[9] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[23]
	leftVal = s[10]
	rightVal = s[11]
	s[10], s[11] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[24]
	leftVal = s[12]
	rightVal = s[13]
	s[12], s[13] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[25]
	leftVal = s[14]
	rightVal = s[15]
	s[14], s[15] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[26]
	leftVal = s[16]
	rightVal = s[17]
	s[16], s[17] = mulPlainLazy2(parentInv, rightVal, parentInv, leftVal)

	// Final layer writes strictly reduced values, eliminating trailing reduce pass.
	parentInv = s[0]
	leftVal = x[0]
	rightVal = x[1]
	x[0], x[1] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[1]
	leftVal = x[2]
	rightVal = x[3]
	x[2], x[3] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[2]
	leftVal = x[4]
	rightVal = x[5]
	x[4], x[5] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[3]
	leftVal = x[6]
	rightVal = x[7]
	x[6], x[7] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[4]
	leftVal = x[8]
	rightVal = x[9]
	x[8], x[9] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[5]
	leftVal = x[10]
	rightVal = x[11]
	x[10], x[11] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[6]
	leftVal = x[12]
	rightVal = x[13]
	x[12], x[13] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[7]
	leftVal = x[14]
	rightVal = x[15]
	x[14], x[15] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[8]
	leftVal = x[16]
	rightVal = x[17]
	x[16], x[17] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[9]
	leftVal = x[18]
	rightVal = x[19]
	x[18], x[19] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[10]
	leftVal = x[20]
	rightVal = x[21]
	x[20], x[21] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[11]
	leftVal = x[22]
	rightVal = x[23]
	x[22], x[23] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[12]
	leftVal = x[24]
	rightVal = x[25]
	x[24], x[25] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[13]
	leftVal = x[26]
	rightVal = x[27]
	x[26], x[27] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[14]
	leftVal = x[28]
	rightVal = x[29]
	x[28], x[29] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[15]
	leftVal = x[30]
	rightVal = x[31]
	x[30], x[31] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	parentInv = s[16]
	leftVal = x[32]
	rightVal = x[33]
	x[32], x[33] = mulPlainStrict2(parentInv, rightVal, parentInv, leftVal)

	x[34] = reduce(s[17])
}

// BatchInvMontTreeNoZeroILP4 is like BatchInvMontTreeNoZero but with 4-pair unrolling

func TestBatchInvTreeNoZeroILP4_35PlainMatches(t *testing.T) {
	for iter := 0; iter < 1000; iter++ {
		var xs1 [PosT]uint32
		var xs2 [PosT]uint32
		for i := 0; i < PosT; i++ {
			v := uint32((iter*131 + i*17 + 1) % int(Q))
			if v == 0 {
				v = 1
			}
			xs1[i] = v
			xs2[i] = v
		}

		BatchInv(xs1[:])
		scratch := make([]uint32, 128)
		batchInvTreeNoZeroILP4_35Plain(xs2[:], scratch)

		for i := 0; i < PosT; i++ {
			if xs1[i] != xs2[i] {
				t.Fatalf("iter=%d idx=%d got=%d want=%d", iter, i, xs2[i], xs1[i])
			}
		}
	}
}

func TestReduceBarrett64LazyBound(t *testing.T) {
	maxP := uint64(4)*uint64(Q)*uint64(Q) - 1
	edges := []uint64{
		0, 1, uint64(Q - 1), uint64(Q), uint64(Q + 1),
		uint64(2*Q - 1), uint64(2 * Q), maxP - 1, maxP,
	}
	for _, p := range edges {
		got := reduceBarrett64Lazy(p)
		if got >= 2*Q {
			t.Fatalf("edge p=%d got=%d out of lazy range", p, got)
		}
		if want := uint32(p % uint64(Q)); reduce(got) != want {
			t.Fatalf("edge p=%d got=%d reduced=%d want=%d", p, got, reduce(got), want)
		}
	}

	x := uint64(1)
	for i := 0; i < 500000; i++ {
		x = x*6364136223846793005 + 1442695040888963407
		p := x & ((uint64(1) << 48) - 1)
		if p > maxP {
			p %= maxP + 1
		}

		got := reduceBarrett64Lazy(p)
		if got >= 2*Q {
			t.Fatalf("rand p=%d got=%d out of lazy range", p, got)
		}
		if want := uint32(p % uint64(Q)); reduce(got) != want {
			t.Fatalf("rand p=%d got=%d reduced=%d want=%d", p, got, reduce(got), want)
		}
	}
}

func TestMulPlainLazyMatches(t *testing.T) {
	x := uint32(1)
	y := uint32(2)
	for i := 0; i < 500000; i++ {
		x = x*1664525 + 1013904223
		y = y*22695477 + 1
		a := x % (2 * Q)
		b := y % (2 * Q)

		got := mulPlainLazy(a, b)
		if got >= 2*Q {
			t.Fatalf("a=%d b=%d got=%d out of lazy range", a, b, got)
		}
		want := uint32((uint64(a) * uint64(b)) % uint64(Q))
		if reduce(got) != want {
			t.Fatalf("a=%d b=%d got=%d reduced=%d want=%d", a, b, got, reduce(got), want)
		}

		gotStrict := mulPlainStrict(a, b)
		if gotStrict != want {
			t.Fatalf("strict a=%d b=%d got=%d want=%d", a, b, gotStrict, want)
		}
	}
}

func TestMulPlainLazy2Matches(t *testing.T) {
	x := uint32(1)
	y := uint32(2)
	for i := 0; i < 300000; i++ {
		x = x*1664525 + 1013904223
		y = y*22695477 + 1
		a0 := x % (2 * Q)
		b0 := y % (2 * Q)

		x = x*1664525 + 1013904223
		y = y*22695477 + 1
		a1 := x % (2 * Q)
		b1 := y % (2 * Q)

		got0, got1 := mulPlainLazy2(a0, b0, a1, b1)
		want0 := uint32((uint64(a0) * uint64(b0)) % uint64(Q))
		want1 := uint32((uint64(a1) * uint64(b1)) % uint64(Q))
		if got0 >= 2*Q || reduce(got0) != want0 {
			t.Fatalf("pair0 a=%d b=%d got=%d reduced=%d want=%d", a0, b0, got0, reduce(got0), want0)
		}
		if got1 >= 2*Q || reduce(got1) != want1 {
			t.Fatalf("pair1 a=%d b=%d got=%d reduced=%d want=%d", a1, b1, got1, reduce(got1), want1)
		}
	}
}

func TestInvPlainLazyMatches(t *testing.T) {
	for v := uint32(1); v < 200000; v++ {
		a := v % Q
		if a == 0 {
			continue
		}
		got := invPlainLazy(a)
		want := Inv(a)
		if got != want {
			t.Fatalf("a=%d got=%d want=%d", a, got, want)
		}
	}
}

func TestBatchInvTreeNoZeroILP4_35PlainLazyMatches(t *testing.T) {
	for iter := 0; iter < 1000; iter++ {
		var xs1 [PosT]uint32
		var xs2 [PosT]uint32
		for i := 0; i < PosT; i++ {
			v := uint32((iter*131 + i*17 + 1) % int(Q))
			if v == 0 {
				v = 1
			}
			xs1[i] = v
			xs2[i] = v
		}

		BatchInv(xs1[:])
		scratch := make([]uint32, 128)
		batchInvTreeNoZeroILP4_35PlainLazy(xs2[:], scratch)

		for i := 0; i < PosT; i++ {
			if xs1[i] != xs2[i] {
				t.Fatalf("iter=%d idx=%d got=%d want=%d", iter, i, xs2[i], xs1[i])
			}
		}
	}
}

func BenchmarkBatchInvTreeNoZeroILP4_35Plain(b *testing.B) {
	xs := make([]uint32, PosT)
	scratch := make([]uint32, 128)
	for i := 0; i < PosT; i++ {
		xs[i] = uint32(i + 1)
	}
	orig := make([]uint32, PosT)
	copy(orig, xs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(xs, orig)
		batchInvTreeNoZeroILP4_35Plain(xs, scratch)
	}
}

func BenchmarkBatchInvTreeNoZeroILP4_35PlainLazy(b *testing.B) {
	xs := make([]uint32, PosT)
	scratch := make([]uint32, 128)
	for i := 0; i < PosT; i++ {
		xs[i] = uint32(i + 1)
	}
	orig := make([]uint32, PosT)
	copy(orig, xs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(xs, orig)
		batchInvTreeNoZeroILP4_35PlainLazy(xs, scratch)
	}
}

func BenchmarkPrimitiveInv(b *testing.B) {
	as, _ := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := as[idx]
		if v == 0 {
			v = 1
		}
		primitiveSink = Inv(v)
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveInvPlainLazy(b *testing.B) {
	as, _ := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := as[idx]
		if v == 0 {
			v = 1
		}
		primitiveSink = invPlainLazy(v)
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveMulPlainLazyInputs(b *testing.B) {
	as, bs := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = mulPlainLazy(as[idx], bs[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}
