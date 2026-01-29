package field

// BatchInvMontTree computes batch modular inverse using tree-based algorithm.
// This converts O(n) sequential depth to O(log n) depth with parallel operations
// within each layer, enabling better instruction-level parallelism.
//
// For n=35: 34 muls up-sweep (6 layers) + 1 inversion + 34 muls down-sweep (6 layers)
// vs sequential: 34 muls forward + 1 inversion + 34 muls backward (all sequential)
//
// The key advantage: within each layer, all multiplications are INDEPENDENT.
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

	// ============ UP-SWEEP ============
	for l := 0; l < maxLayers; l++ {
		srcOff := layerOff[l]
		srcCnt := layerCnt[l]
		dstOff := layerOff[l+1]

		pairs := srcCnt / 2
		for p := 0; p < pairs; p++ {
			scratch[dstOff+p] = mulMontLazy(scratch[srcOff+p*2], scratch[srcOff+p*2+1])
		}

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

		for p := 0; p < pairs; p++ {
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
// If no zeros exist (common case), uses the faster NoZero path.
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
		BatchInvMontTreeNoZero(xs, scratch)
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
