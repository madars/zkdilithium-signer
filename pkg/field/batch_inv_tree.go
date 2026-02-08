package field

// BatchInvMontTree computes batch modular inverse using tree-based algorithm.
// This converts O(n) sequential depth to O(log n) depth with parallel operations
// within each layer, enabling better instruction-level parallelism.
//
// For n=35: 34 muls up-sweep (6 layers) + 1 inversion + 34 muls down-sweep (6 layers)
// vs sequential: 34 muls forward + 1 inversion + 34 muls backward (all sequential)
//
// The key advantage: within each layer, all multiplications are INDEPENDENT.
// Uses 4-pair unrolling in up-sweep and down-sweep for better ILP.
// scratch must have capacity >= 3*n.
func BatchInvMontTree(xs []uint32, scratch []uint32) {
	n := len(xs)
	if n == 0 {
		return
	}
	if n == 1 {
		if xs[0] != 0 {
			xs[0] = InvMont(reduce(xs[0]))
		}
		return
	}

	oneM := ToMont(1)

	// Copy inputs to working buffer, handling zeros
	work := scratch[:n]
	for i := 0; i < n; i++ {
		x := xs[i]
		if x == 0 {
			work[i] = oneM // Use 1_M for zeros
		} else {
			work[i] = x
		}
	}

	// Calculate layers needed (max 8 for n<=256, 10 for n<=1024)
	maxLayers := 0
	for temp := n; temp > 1; temp = (temp + 1) / 2 {
		maxLayers++
	}

	// Layer storage: fixed-size array to avoid allocation
	var layerOff [16]int
	var layerCnt [16]int

	layerOff[0] = 0
	layerCnt[0] = n

	offset := n
	currentCount := n
	for l := 1; l <= maxLayers; l++ {
		nextCount := (currentCount + 1) / 2
		layerOff[l] = offset
		layerCnt[l] = nextCount
		offset += nextCount
		currentCount = nextCount
	}

	// ============ UP-SWEEP with 4-pair unrolling ============
	for l := 0; l < maxLayers; l++ {
		srcOff := layerOff[l]
		srcCnt := layerCnt[l]
		dstOff := layerOff[l+1]
		pairs := srcCnt / 2

		p := 0
		for ; p+3 < pairs; p += 4 {
			s0 := scratch[srcOff+p*2]
			s1 := scratch[srcOff+p*2+1]
			s2 := scratch[srcOff+p*2+2]
			s3 := scratch[srcOff+p*2+3]
			s4 := scratch[srcOff+p*2+4]
			s5 := scratch[srcOff+p*2+5]
			s6 := scratch[srcOff+p*2+6]
			s7 := scratch[srcOff+p*2+7]
			scratch[dstOff+p] = mulMontLazy(s0, s1)
			scratch[dstOff+p+1] = mulMontLazy(s2, s3)
			scratch[dstOff+p+2] = mulMontLazy(s4, s5)
			scratch[dstOff+p+3] = mulMontLazy(s6, s7)
		}
		for ; p < pairs; p++ {
			scratch[dstOff+p] = mulMontLazy(scratch[srcOff+p*2], scratch[srcOff+p*2+1])
		}
		if srcCnt%2 == 1 {
			scratch[dstOff+pairs] = scratch[srcOff+srcCnt-1]
		}
	}

	// ============ INVERT ROOT ============
	rootOff := layerOff[maxLayers]
	scratch[rootOff] = InvMont(reduce(scratch[rootOff]))

	// ============ DOWN-SWEEP with 4-pair unrolling ============
	for l := maxLayers; l > 0; l-- {
		parentOff := layerOff[l]
		childOff := layerOff[l-1]
		childCnt := layerCnt[l-1]
		pairs := childCnt / 2

		p := 0
		for ; p+3 < pairs; p += 4 {
			p1 := scratch[parentOff+p]
			p2 := scratch[parentOff+p+1]
			p3 := scratch[parentOff+p+2]
			p4 := scratch[parentOff+p+3]
			l1 := scratch[childOff+p*2]
			r1 := scratch[childOff+p*2+1]
			l2 := scratch[childOff+p*2+2]
			r2 := scratch[childOff+p*2+3]
			l3 := scratch[childOff+p*2+4]
			r3 := scratch[childOff+p*2+5]
			l4 := scratch[childOff+p*2+6]
			r4 := scratch[childOff+p*2+7]

			scratch[childOff+p*2] = mulMontLazy(p1, r1)
			scratch[childOff+p*2+1] = mulMontLazy(p1, l1)
			scratch[childOff+p*2+2] = mulMontLazy(p2, r2)
			scratch[childOff+p*2+3] = mulMontLazy(p2, l2)
			scratch[childOff+p*2+4] = mulMontLazy(p3, r3)
			scratch[childOff+p*2+5] = mulMontLazy(p3, l3)
			scratch[childOff+p*2+6] = mulMontLazy(p4, r4)
			scratch[childOff+p*2+7] = mulMontLazy(p4, l4)
		}
		for ; p < pairs; p++ {
			parentInv := scratch[parentOff+p]
			leftVal := scratch[childOff+p*2]
			rightVal := scratch[childOff+p*2+1]
			scratch[childOff+p*2] = mulMontLazy(parentInv, rightVal)
			scratch[childOff+p*2+1] = mulMontLazy(parentInv, leftVal)
		}
		if childCnt%2 == 1 {
			scratch[childOff+childCnt-1] = scratch[parentOff+pairs]
		}
	}

	// ============ WRITE BACK ============
	for i := 0; i < n; i++ {
		if xs[i] == 0 {
			continue // Zero stays zero
		}
		xs[i] = reduce(work[i])
	}
}

// BatchInvMontTreeCond checks for zeros first and dispatches to the appropriate version.
// If no zeros exist (common case), uses the faster NoZeroILP4 path with 4-pair unrolling.
func BatchInvMontTreeCond(xs []uint32, scratch []uint32) {
	n := len(xs)
	hasZero := false
	for i := 0; i < n; i++ {
		if xs[i] == 0 {
			hasZero = true
			break
		}
	}
	if hasZero {
		BatchInvMontTree(xs, scratch)
	} else {
		BatchInvMontTreeNoZeroILP4(xs, scratch)
	}
}

// batchInvMontTreeNoZeroILP4_35 is a fixed-size fast path for Poseidon width (n=35).
// It removes dynamic layer setup and keeps the same 4-pair ILP structure.
func batchInvMontTreeNoZeroILP4_35(xs []uint32, scratch []uint32) {
	x := (*[PosT]uint32)(xs)   // 35
	s := (*[38]uint32)(scratch) // 18+9+5+3+2+1

	// ============ UP-SWEEP ============
	s[0] = mulMontLazy(x[0], x[1])
	s[1] = mulMontLazy(x[2], x[3])
	s[2] = mulMontLazy(x[4], x[5])
	s[3] = mulMontLazy(x[6], x[7])
	s[4] = mulMontLazy(x[8], x[9])
	s[5] = mulMontLazy(x[10], x[11])
	s[6] = mulMontLazy(x[12], x[13])
	s[7] = mulMontLazy(x[14], x[15])
	s[8] = mulMontLazy(x[16], x[17])
	s[9] = mulMontLazy(x[18], x[19])
	s[10] = mulMontLazy(x[20], x[21])
	s[11] = mulMontLazy(x[22], x[23])
	s[12] = mulMontLazy(x[24], x[25])
	s[13] = mulMontLazy(x[26], x[27])
	s[14] = mulMontLazy(x[28], x[29])
	s[15] = mulMontLazy(x[30], x[31])
	s[16] = mulMontLazy(x[32], x[33])
	s[17] = x[34]

	s[18] = mulMontLazy(s[0], s[1])
	s[19] = mulMontLazy(s[2], s[3])
	s[20] = mulMontLazy(s[4], s[5])
	s[21] = mulMontLazy(s[6], s[7])
	s[22] = mulMontLazy(s[8], s[9])
	s[23] = mulMontLazy(s[10], s[11])
	s[24] = mulMontLazy(s[12], s[13])
	s[25] = mulMontLazy(s[14], s[15])
	s[26] = mulMontLazy(s[16], s[17])

	s[27] = mulMontLazy(s[18], s[19])
	s[28] = mulMontLazy(s[20], s[21])
	s[29] = mulMontLazy(s[22], s[23])
	s[30] = mulMontLazy(s[24], s[25])
	s[31] = s[26]

	s[32] = mulMontLazy(s[27], s[28])
	s[33] = mulMontLazy(s[29], s[30])
	s[34] = s[31]

	s[35] = mulMontLazy(s[32], s[33])
	s[36] = s[34]

	// ============ INVERT ROOT ============
	s[37] = InvMont(reduce(mulMontLazy(s[35], s[36])))

	// ============ DOWN-SWEEP ============
	parentInv := s[37]
	leftVal := s[35]
	rightVal := s[36]
	s[35] = mulMontLazy(parentInv, rightVal)
	s[36] = mulMontLazy(parentInv, leftVal)

	parentInv = s[35]
	leftVal = s[32]
	rightVal = s[33]
	s[32] = mulMontLazy(parentInv, rightVal)
	s[33] = mulMontLazy(parentInv, leftVal)
	s[34] = s[36]

	parentInv = s[32]
	leftVal = s[27]
	rightVal = s[28]
	s[27] = mulMontLazy(parentInv, rightVal)
	s[28] = mulMontLazy(parentInv, leftVal)

	parentInv = s[33]
	leftVal = s[29]
	rightVal = s[30]
	s[29] = mulMontLazy(parentInv, rightVal)
	s[30] = mulMontLazy(parentInv, leftVal)
	s[31] = s[34]

	parentInv = s[27]
	leftVal = s[18]
	rightVal = s[19]
	s[18] = mulMontLazy(parentInv, rightVal)
	s[19] = mulMontLazy(parentInv, leftVal)

	parentInv = s[28]
	leftVal = s[20]
	rightVal = s[21]
	s[20] = mulMontLazy(parentInv, rightVal)
	s[21] = mulMontLazy(parentInv, leftVal)

	parentInv = s[29]
	leftVal = s[22]
	rightVal = s[23]
	s[22] = mulMontLazy(parentInv, rightVal)
	s[23] = mulMontLazy(parentInv, leftVal)

	parentInv = s[30]
	leftVal = s[24]
	rightVal = s[25]
	s[24] = mulMontLazy(parentInv, rightVal)
	s[25] = mulMontLazy(parentInv, leftVal)
	s[26] = s[31]

	parentInv = s[18]
	leftVal = s[0]
	rightVal = s[1]
	s[0] = mulMontLazy(parentInv, rightVal)
	s[1] = mulMontLazy(parentInv, leftVal)

	parentInv = s[19]
	leftVal = s[2]
	rightVal = s[3]
	s[2] = mulMontLazy(parentInv, rightVal)
	s[3] = mulMontLazy(parentInv, leftVal)

	parentInv = s[20]
	leftVal = s[4]
	rightVal = s[5]
	s[4] = mulMontLazy(parentInv, rightVal)
	s[5] = mulMontLazy(parentInv, leftVal)

	parentInv = s[21]
	leftVal = s[6]
	rightVal = s[7]
	s[6] = mulMontLazy(parentInv, rightVal)
	s[7] = mulMontLazy(parentInv, leftVal)

	parentInv = s[22]
	leftVal = s[8]
	rightVal = s[9]
	s[8] = mulMontLazy(parentInv, rightVal)
	s[9] = mulMontLazy(parentInv, leftVal)

	parentInv = s[23]
	leftVal = s[10]
	rightVal = s[11]
	s[10] = mulMontLazy(parentInv, rightVal)
	s[11] = mulMontLazy(parentInv, leftVal)

	parentInv = s[24]
	leftVal = s[12]
	rightVal = s[13]
	s[12] = mulMontLazy(parentInv, rightVal)
	s[13] = mulMontLazy(parentInv, leftVal)

	parentInv = s[25]
	leftVal = s[14]
	rightVal = s[15]
	s[14] = mulMontLazy(parentInv, rightVal)
	s[15] = mulMontLazy(parentInv, leftVal)

	parentInv = s[26]
	leftVal = s[16]
	rightVal = s[17]
	s[16] = mulMontLazy(parentInv, rightVal)
	s[17] = mulMontLazy(parentInv, leftVal)

	// Final layer writes strictly reduced values, eliminating trailing reduce pass.
	parentInv = s[0]
	leftVal = x[0]
	rightVal = x[1]
	x[0] = MulMont(parentInv, rightVal)
	x[1] = MulMont(parentInv, leftVal)

	parentInv = s[1]
	leftVal = x[2]
	rightVal = x[3]
	x[2] = MulMont(parentInv, rightVal)
	x[3] = MulMont(parentInv, leftVal)

	parentInv = s[2]
	leftVal = x[4]
	rightVal = x[5]
	x[4] = MulMont(parentInv, rightVal)
	x[5] = MulMont(parentInv, leftVal)

	parentInv = s[3]
	leftVal = x[6]
	rightVal = x[7]
	x[6] = MulMont(parentInv, rightVal)
	x[7] = MulMont(parentInv, leftVal)

	parentInv = s[4]
	leftVal = x[8]
	rightVal = x[9]
	x[8] = MulMont(parentInv, rightVal)
	x[9] = MulMont(parentInv, leftVal)

	parentInv = s[5]
	leftVal = x[10]
	rightVal = x[11]
	x[10] = MulMont(parentInv, rightVal)
	x[11] = MulMont(parentInv, leftVal)

	parentInv = s[6]
	leftVal = x[12]
	rightVal = x[13]
	x[12] = MulMont(parentInv, rightVal)
	x[13] = MulMont(parentInv, leftVal)

	parentInv = s[7]
	leftVal = x[14]
	rightVal = x[15]
	x[14] = MulMont(parentInv, rightVal)
	x[15] = MulMont(parentInv, leftVal)

	parentInv = s[8]
	leftVal = x[16]
	rightVal = x[17]
	x[16] = MulMont(parentInv, rightVal)
	x[17] = MulMont(parentInv, leftVal)

	parentInv = s[9]
	leftVal = x[18]
	rightVal = x[19]
	x[18] = MulMont(parentInv, rightVal)
	x[19] = MulMont(parentInv, leftVal)

	parentInv = s[10]
	leftVal = x[20]
	rightVal = x[21]
	x[20] = MulMont(parentInv, rightVal)
	x[21] = MulMont(parentInv, leftVal)

	parentInv = s[11]
	leftVal = x[22]
	rightVal = x[23]
	x[22] = MulMont(parentInv, rightVal)
	x[23] = MulMont(parentInv, leftVal)

	parentInv = s[12]
	leftVal = x[24]
	rightVal = x[25]
	x[24] = MulMont(parentInv, rightVal)
	x[25] = MulMont(parentInv, leftVal)

	parentInv = s[13]
	leftVal = x[26]
	rightVal = x[27]
	x[26] = MulMont(parentInv, rightVal)
	x[27] = MulMont(parentInv, leftVal)

	parentInv = s[14]
	leftVal = x[28]
	rightVal = x[29]
	x[28] = MulMont(parentInv, rightVal)
	x[29] = MulMont(parentInv, leftVal)

	parentInv = s[15]
	leftVal = x[30]
	rightVal = x[31]
	x[30] = MulMont(parentInv, rightVal)
	x[31] = MulMont(parentInv, leftVal)

	parentInv = s[16]
	leftVal = x[32]
	rightVal = x[33]
	x[32] = MulMont(parentInv, rightVal)
	x[33] = MulMont(parentInv, leftVal)

	x[34] = reduce(s[17])
}

// BatchInvMontTreeNoZeroILP4 is like BatchInvMontTreeNoZero but with 4-pair unrolling
// in up-sweep and down-sweep for better instruction-level parallelism.
func BatchInvMontTreeNoZeroILP4(xs []uint32, scratch []uint32) {
	n := len(xs)
	if n == 0 {
		return
	}
	if n == 1 {
		xs[0] = InvMont(reduce(xs[0]))
		return
	}
	if n == PosT {
		batchInvMontTreeNoZeroILP4_35(xs, scratch)
		return
	}

	work := scratch[:n]
	copy(work, xs)

	maxLayers := 0
	for temp := n; temp > 1; temp = (temp + 1) / 2 {
		maxLayers++
	}

	var layerOff [16]int
	var layerCnt [16]int

	layerOff[0] = 0
	layerCnt[0] = n

	offset := n
	currentCount := n
	for l := 1; l <= maxLayers; l++ {
		nextCount := (currentCount + 1) / 2
		layerOff[l] = offset
		layerCnt[l] = nextCount
		offset += nextCount
		currentCount = nextCount
	}

	// ============ UP-SWEEP with 4-pair unrolling ============
	for l := 0; l < maxLayers; l++ {
		srcOff := layerOff[l]
		srcCnt := layerCnt[l]
		dstOff := layerOff[l+1]
		pairs := srcCnt / 2

		p := 0
		for ; p+3 < pairs; p += 4 {
			s0 := scratch[srcOff+p*2]
			s1 := scratch[srcOff+p*2+1]
			s2 := scratch[srcOff+p*2+2]
			s3 := scratch[srcOff+p*2+3]
			s4 := scratch[srcOff+p*2+4]
			s5 := scratch[srcOff+p*2+5]
			s6 := scratch[srcOff+p*2+6]
			s7 := scratch[srcOff+p*2+7]
			scratch[dstOff+p] = mulMontLazy(s0, s1)
			scratch[dstOff+p+1] = mulMontLazy(s2, s3)
			scratch[dstOff+p+2] = mulMontLazy(s4, s5)
			scratch[dstOff+p+3] = mulMontLazy(s6, s7)
		}
		for ; p < pairs; p++ {
			scratch[dstOff+p] = mulMontLazy(scratch[srcOff+p*2], scratch[srcOff+p*2+1])
		}
		if srcCnt%2 == 1 {
			scratch[dstOff+pairs] = scratch[srcOff+srcCnt-1]
		}
	}

	// ============ INVERT ROOT ============
	rootOff := layerOff[maxLayers]
	scratch[rootOff] = InvMont(reduce(scratch[rootOff]))

	// ============ DOWN-SWEEP with 4-pair unrolling ============
	for l := maxLayers; l > 0; l-- {
		parentOff := layerOff[l]
		childOff := layerOff[l-1]
		childCnt := layerCnt[l-1]
		pairs := childCnt / 2

		p := 0
		for ; p+3 < pairs; p += 4 {
			p1 := scratch[parentOff+p]
			p2 := scratch[parentOff+p+1]
			p3 := scratch[parentOff+p+2]
			p4 := scratch[parentOff+p+3]
			l1 := scratch[childOff+p*2]
			r1 := scratch[childOff+p*2+1]
			l2 := scratch[childOff+p*2+2]
			r2 := scratch[childOff+p*2+3]
			l3 := scratch[childOff+p*2+4]
			r3 := scratch[childOff+p*2+5]
			l4 := scratch[childOff+p*2+6]
			r4 := scratch[childOff+p*2+7]

			scratch[childOff+p*2] = mulMontLazy(p1, r1)
			scratch[childOff+p*2+1] = mulMontLazy(p1, l1)
			scratch[childOff+p*2+2] = mulMontLazy(p2, r2)
			scratch[childOff+p*2+3] = mulMontLazy(p2, l2)
			scratch[childOff+p*2+4] = mulMontLazy(p3, r3)
			scratch[childOff+p*2+5] = mulMontLazy(p3, l3)
			scratch[childOff+p*2+6] = mulMontLazy(p4, r4)
			scratch[childOff+p*2+7] = mulMontLazy(p4, l4)
		}
		for ; p < pairs; p++ {
			parentInv := scratch[parentOff+p]
			leftVal := scratch[childOff+p*2]
			rightVal := scratch[childOff+p*2+1]
			scratch[childOff+p*2] = mulMontLazy(parentInv, rightVal)
			scratch[childOff+p*2+1] = mulMontLazy(parentInv, leftVal)
		}
		if childCnt%2 == 1 {
			scratch[childOff+childCnt-1] = scratch[parentOff+pairs]
		}
	}

	// ============ WRITE BACK ============
	for i := 0; i < n; i++ {
		xs[i] = reduce(work[i])
	}
}

// BatchInvMontTreeNoZeroILP is like BatchInvMontTreeNoZero but with 2-pair unrolling
// in up-sweep and down-sweep for better instruction-level parallelism.
func BatchInvMontTreeNoZeroILP(xs []uint32, scratch []uint32) {
	n := len(xs)
	if n == 0 {
		return
	}
	if n == 1 {
		xs[0] = InvMont(reduce(xs[0]))
		return
	}

	// Copy to scratch for tree building
	work := scratch[:n]
	copy(work, xs)

	// Calculate layers needed
	maxLayers := 0
	for temp := n; temp > 1; temp = (temp + 1) / 2 {
		maxLayers++
	}

	var layerOff [16]int
	var layerCnt [16]int

	layerOff[0] = 0
	layerCnt[0] = n

	offset := n
	currentCount := n
	for l := 1; l <= maxLayers; l++ {
		nextCount := (currentCount + 1) / 2
		layerOff[l] = offset
		layerCnt[l] = nextCount
		offset += nextCount
		currentCount = nextCount
	}

	// ============ UP-SWEEP with 2-pair unrolling ============
	for l := 0; l < maxLayers; l++ {
		srcOff := layerOff[l]
		srcCnt := layerCnt[l]
		dstOff := layerOff[l+1]
		pairs := srcCnt / 2

		// Process 2 pairs at a time for ILP
		p := 0
		for ; p+1 < pairs; p += 2 {
			// Load 4 source elements
			s0 := scratch[srcOff+p*2]
			s1 := scratch[srcOff+p*2+1]
			s2 := scratch[srcOff+p*2+2]
			s3 := scratch[srcOff+p*2+3]
			// 2 independent multiplications
			scratch[dstOff+p] = mulMontLazy(s0, s1)
			scratch[dstOff+p+1] = mulMontLazy(s2, s3)
		}
		// Handle remaining pair
		if p < pairs {
			scratch[dstOff+p] = mulMontLazy(scratch[srcOff+p*2], scratch[srcOff+p*2+1])
		}
		// Straggler
		if srcCnt%2 == 1 {
			scratch[dstOff+pairs] = scratch[srcOff+srcCnt-1]
		}
	}

	// ============ INVERT ROOT ============
	rootOff := layerOff[maxLayers]
	scratch[rootOff] = InvMont(reduce(scratch[rootOff]))

	// ============ DOWN-SWEEP with 2-pair unrolling ============
	for l := maxLayers; l > 0; l-- {
		parentOff := layerOff[l]
		childOff := layerOff[l-1]
		childCnt := layerCnt[l-1]
		pairs := childCnt / 2

		// Process 2 pairs at a time for ILP
		p := 0
		for ; p+1 < pairs; p += 2 {
			// Load 2 parents, 4 children
			p1 := scratch[parentOff+p]
			p2 := scratch[parentOff+p+1]
			l1 := scratch[childOff+p*2]
			r1 := scratch[childOff+p*2+1]
			l2 := scratch[childOff+p*2+2]
			r2 := scratch[childOff+p*2+3]

			// 4 independent multiplications
			scratch[childOff+p*2] = mulMontLazy(p1, r1)
			scratch[childOff+p*2+1] = mulMontLazy(p1, l1)
			scratch[childOff+p*2+2] = mulMontLazy(p2, r2)
			scratch[childOff+p*2+3] = mulMontLazy(p2, l2)
		}
		// Handle remaining pair
		if p < pairs {
			parentInv := scratch[parentOff+p]
			leftVal := scratch[childOff+p*2]
			rightVal := scratch[childOff+p*2+1]
			scratch[childOff+p*2] = mulMontLazy(parentInv, rightVal)
			scratch[childOff+p*2+1] = mulMontLazy(parentInv, leftVal)
		}
		// Straggler
		if childCnt%2 == 1 {
			scratch[childOff+childCnt-1] = scratch[parentOff+pairs]
		}
	}

	// ============ WRITE BACK ============
	for i := 0; i < n; i++ {
		xs[i] = reduce(work[i])
	}
}

// BatchInvMontTreeNoZero is optimized version assuming no zeros in input.
// This is the common case for Poseidon S-box where inputs are field elements.
// scratch must have capacity >= 2*n.
func BatchInvMontTreeNoZero(xs []uint32, scratch []uint32) {
	n := len(xs)
	if n == 0 {
		return
	}
	if n == 1 {
		xs[0] = InvMont(reduce(xs[0]))
		return
	}

	// Copy to scratch for tree building
	work := scratch[:n]
	copy(work, xs)

	// Calculate layers needed (max 8 for n<=256, 10 for n<=1024)
	maxLayers := 0
	for temp := n; temp > 1; temp = (temp + 1) / 2 {
		maxLayers++
	}

	// Layer storage: fixed-size array to avoid allocation
	// layerOff[l] = offset in scratch where layer l starts
	// layerCnt[l] = number of elements in layer l
	var layerOff [16]int
	var layerCnt [16]int

	layerOff[0] = 0
	layerCnt[0] = n

	offset := n
	currentCount := n
	for l := 1; l <= maxLayers; l++ {
		nextCount := (currentCount + 1) / 2
		layerOff[l] = offset
		layerCnt[l] = nextCount
		offset += nextCount
		currentCount = nextCount
	}

	// ============ UP-SWEEP ============
	for l := 0; l < maxLayers; l++ {
		srcOff := layerOff[l]
		srcCnt := layerCnt[l]
		dstOff := layerOff[l+1]

		// Process pairs - these are INDEPENDENT within the layer!
		pairs := srcCnt / 2
		for p := 0; p < pairs; p++ {
			scratch[dstOff+p] = mulMontLazy(scratch[srcOff+p*2], scratch[srcOff+p*2+1])
		}

		// Straggler
		if srcCnt%2 == 1 {
			scratch[dstOff+pairs] = scratch[srcOff+srcCnt-1]
		}
	}

	// ============ INVERT ROOT ============
	rootOff := layerOff[maxLayers]
	scratch[rootOff] = InvMont(reduce(scratch[rootOff]))

	// ============ DOWN-SWEEP ============
	for l := maxLayers; l > 0; l-- {
		parentOff := layerOff[l]
		childOff := layerOff[l-1]
		childCnt := layerCnt[l-1]
		pairs := childCnt / 2

		// Process all pairs - INDEPENDENT operations!
		for p := 0; p < pairs; p++ {
			parentInv := scratch[parentOff+p]
			leftVal := scratch[childOff+p*2]
			rightVal := scratch[childOff+p*2+1]

			// Cross-multiply
			scratch[childOff+p*2] = mulMontLazy(parentInv, rightVal)
			scratch[childOff+p*2+1] = mulMontLazy(parentInv, leftVal)
		}

		// Straggler
		if childCnt%2 == 1 {
			scratch[childOff+childCnt-1] = scratch[parentOff+pairs]
		}
	}

	// ============ WRITE BACK ============
	for i := 0; i < n; i++ {
		xs[i] = reduce(work[i])
	}
}
