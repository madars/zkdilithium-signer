package field

// batchInvTreeNoZeroILP4_35PlainLazyProd is a plain-domain batch inversion
// specialized for n=35. It keeps intermediates lazy and writes strict outputs.
func batchInvTreeNoZeroILP4_35PlainLazyProd(xs []uint32, scratch []uint32) bool {
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
	if rootProd == 0 {
		return false
	}
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

	// Final layer writes strictly reduced values.
	// Use mulPlainStrict2 (compiler CSEL) instead of mulPlainLazy2+reduce2 (manual mask).
	parentInv = s[0]
	x[0], x[1] = mulPlainStrict2(parentInv, x[1], parentInv, x[0])

	parentInv = s[1]
	x[2], x[3] = mulPlainStrict2(parentInv, x[3], parentInv, x[2])

	parentInv = s[2]
	x[4], x[5] = mulPlainStrict2(parentInv, x[5], parentInv, x[4])

	parentInv = s[3]
	x[6], x[7] = mulPlainStrict2(parentInv, x[7], parentInv, x[6])

	parentInv = s[4]
	x[8], x[9] = mulPlainStrict2(parentInv, x[9], parentInv, x[8])

	parentInv = s[5]
	x[10], x[11] = mulPlainStrict2(parentInv, x[11], parentInv, x[10])

	parentInv = s[6]
	x[12], x[13] = mulPlainStrict2(parentInv, x[13], parentInv, x[12])

	parentInv = s[7]
	x[14], x[15] = mulPlainStrict2(parentInv, x[15], parentInv, x[14])

	parentInv = s[8]
	x[16], x[17] = mulPlainStrict2(parentInv, x[17], parentInv, x[16])

	parentInv = s[9]
	x[18], x[19] = mulPlainStrict2(parentInv, x[19], parentInv, x[18])

	parentInv = s[10]
	x[20], x[21] = mulPlainStrict2(parentInv, x[21], parentInv, x[20])

	parentInv = s[11]
	x[22], x[23] = mulPlainStrict2(parentInv, x[23], parentInv, x[22])

	parentInv = s[12]
	x[24], x[25] = mulPlainStrict2(parentInv, x[25], parentInv, x[24])

	parentInv = s[13]
	x[26], x[27] = mulPlainStrict2(parentInv, x[27], parentInv, x[26])

	parentInv = s[14]
	x[28], x[29] = mulPlainStrict2(parentInv, x[29], parentInv, x[28])

	parentInv = s[15]
	x[30], x[31] = mulPlainStrict2(parentInv, x[31], parentInv, x[30])

	parentInv = s[16]
	x[32], x[33] = mulPlainStrict2(parentInv, x[33], parentInv, x[32])

	x[34] = reduce(s[17])
	return true
}

// batchInvTreeWithZeroILP4_35PlainLazyProd handles zeros via the standard
// "replace zero with one" trick on a local working copy, while preserving zero
// outputs at the corresponding positions.
func batchInvTreeWithZeroILP4_35PlainLazyProd(xs []uint32, scratch []uint32) {
	work := scratch[:PosT]
	treeScratch := scratch[PosT:]

	for i := 0; i < PosT; i++ {
		v := xs[i]
		if v == 0 {
			work[i] = 1
		} else {
			work[i] = v
		}
	}

	_ = batchInvTreeNoZeroILP4_35PlainLazyProd(work, treeScratch)

	for i := 0; i < PosT; i++ {
		if xs[i] != 0 {
			xs[i] = work[i]
		}
	}
}

// BatchInvTreeCondPlain performs batch inversion in plain field representation.
// Zero entries remain zero.
func BatchInvTreeCondPlain(xs []uint32, scratch []uint32) {
	n := len(xs)
	if n == 0 {
		return
	}
	if n == 1 {
		if xs[0] != 0 {
			xs[0] = Inv(xs[0])
		}
		return
	}

	if n == PosT {
		if !batchInvTreeNoZeroILP4_35PlainLazyProd(xs, scratch) {
			batchInvTreeWithZeroILP4_35PlainLazyProd(xs, scratch)
		}
		return
	}

	hasZero := false
	for i := 0; i < n; i++ {
		if xs[i] == 0 {
			hasZero = true
			break
		}
	}

	if hasZero {
		BatchInv(xs)
		return
	}

	BatchInv(xs)
}
