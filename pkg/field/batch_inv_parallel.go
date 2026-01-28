package field

// BatchInvMontParallel computes batch modular inverse with ILP optimization.
// Processes pairs of elements to enable instruction-level parallelism.
// Uses branchless zero handling to keep operations uniform.
//
// Key ideas:
// 1. Branchless mask: nonZeroMask = (x | -x) >> 63
// 2. Process pairs: start both MULs before completing reductions
// 3. Uniform operations enable better pipelining
func BatchInvMontParallel(xs []uint32, scratch []uint32) {
	n := len(xs)
	if n == 0 {
		return
	}

	prods := scratch[:n]
	oneM := uint64(ToMont(1))

	// Forward pass: compute prefix products (branchless)
	{
		x := uint64(xs[0])
		nzm := (x | -x) >> 63 // 1 if nonzero
		zm := 1 ^ nzm         // 1 if zero
		prods[0] = uint32(x*nzm + oneM*zm)
	}

	// Process pairs in forward pass
	i := 1
	for i+1 < n {
		// Load pair
		x0 := uint64(xs[i])
		x1 := uint64(xs[i+1])

		// Branchless masks for both
		nzm0 := (x0 | -x0) >> 63
		nzm1 := (x1 | -x1) >> 63
		zm0 := 1 ^ nzm0
		zm1 := 1 ^ nzm1

		// Safe values (x if nonzero, 1_M if zero)
		safe0 := uint32(x0*nzm0 + oneM*zm0)
		safe1 := uint32(x1*nzm1 + oneM*zm1)

		// First multiplication
		prev := prods[i-1]
		prods[i] = mulMontLazy(prev, safe0)

		// Second multiplication (depends on first)
		prods[i+1] = mulMontLazy(prods[i], safe1)

		i += 2
	}

	// Handle odd element
	if i < n {
		x := uint64(xs[i])
		nzm := (x | -x) >> 63
		zm := 1 ^ nzm
		safe := uint32(x*nzm + oneM*zm)
		prods[i] = mulMontLazy(prods[i-1], safe)
	}

	// Invert final product
	inv := InvMont(reduce(prods[n-1]))

	// Backward pass: compute individual inverses
	// Process pairs for ILP
	j := n - 1
	for j >= 2 {
		// Load pair of original values
		x0 := uint64(xs[j])
		x1 := uint64(xs[j-1])

		// Branchless masks
		nzm0 := (x0 | -x0) >> 63
		nzm1 := (x1 | -x1) >> 63
		zm0 := 1 ^ nzm0
		zm1 := 1 ^ nzm1

		// Safe values for inv update
		safe0 := uint32(x0*nzm0 + oneM*zm0)
		safe1 := uint32(x1*nzm1 + oneM*zm1)

		// Compute results (interleaved for ILP)
		// Start both multiplications
		t0 := uint64(inv) * uint64(prods[j-1])
		inv1 := mulMontLazy(inv, safe0) // inv for next iteration
		t1 := uint64(inv1) * uint64(prods[j-2])

		// Complete Montgomery reductions
		res0 := montRedLazy(t0)
		res1 := montRedLazy(t1)

		// Mask results and store
		xs[j] = uint32(uint64(res0) * nzm0)
		xs[j-1] = uint32(uint64(res1) * nzm1)

		// Update inv
		inv = mulMontLazy(inv1, safe1)

		j -= 2
	}

	// Handle remaining element(s)
	if j == 1 {
		x := uint64(xs[1])
		nzm := (x | -x) >> 63
		zm := 1 ^ nzm
		safe := uint32(x*nzm + oneM*zm)

		res := mulMontLazy(inv, prods[0])
		xs[1] = uint32(uint64(res) * nzm)
		inv = mulMontLazy(inv, safe)
	}

	// Handle xs[0]
	{
		x := uint64(xs[0])
		nzm := (x | -x) >> 63
		xs[0] = uint32(uint64(inv) * nzm)
	}

	// Final reduction - can process pairs
	k := 0
	for k+1 < n {
		xs[k] = reduce(xs[k])
		xs[k+1] = reduce(xs[k+1])
		k += 2
	}
	if k < n {
		xs[k] = reduce(xs[k])
	}
}

// montRedLazy performs lazy Montgomery reduction on a uint64 product.
// Returns result < 2Q.
func montRedLazy(t uint64) uint32 {
	const montgomeryQInvNeg uint32 = 7340031
	m := uint32(t) * montgomeryQInvNeg
	u := (t + uint64(m)*Q) >> 32
	return uint32(u)
}
