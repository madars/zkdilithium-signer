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
