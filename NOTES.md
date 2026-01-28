# Development Notes

## Approach: Python to Go Translation via Test Vectors

This project was translated from the [reference Python implementation](https://github.com/guruvamsi-policharla/zkdilithium) using a test-vector-driven approach:

1. **Generate deterministic test vectors from Python** - Using fixed seeds, generate expected outputs for every function (NTT, Poseidon, sampling, signing, etc.)

2. **Implement Go functions one at a time** - Start with primitives (field arithmetic), work up to high-level operations (Sign/Verify)

3. **Verify byte-for-byte match** - Each Go function must produce identical output to Python for the same input

4. **Gold standard test** - The ultimate validation is that `Sign(sk, "test")` produces the exact same 2340-byte signature as Python

Why bottom-up with test vectors at each level?
- End-to-end tests alone are impossible to debug - if Sign() fails, where's the bug?
- Test vectors at each level isolate bugs: if NTT passes but PolyMul fails, bug is in PolyMul
- Catches subtle issues: off-by-one errors, reduction conventions, endianness

## Performance Optimization Journey

### Starting Point
- Sign: 19.8ms
- Verify: 2.9ms
- Allocations: ~7000+ per Sign

### Final Results
- Sign: 6.0ms (3.3x faster)
- Verify: 1.0ms (2.9x faster)
- Allocations: ~110 per Sign

### Optimizations Applied (in commit order)

#### 1. Optimized Modular Inverse (addition chain)
Uses a fixed addition chain for `a^(Q-2)` exploiting `Q-2 = 0b110_11111111111111111111`.
Reduces operations from ~43 to 30 per `Inv()` call.

#### 2. Batch Inversion for Poseidon S-box
Montgomery's trick: n inversions with 1 inversion + 3(n-1) multiplications.
The Poseidon S-box does 35 inversions per round × 21 rounds = 735 inversions per permutation.
With batch inversion: 21 inversions + ~2000 multiplications (much faster).

#### 3. Montgomery Multiplication for NTT
Precompute zetas in Montgomery form. NTT inner loop uses `MulMont` instead of `Mul`.
~36% improvement in isolated NTT benchmark.

#### 4. Full Montgomery Form Throughout
All NTT and Poseidon operations stay in Montgomery form. Convert to/from Montgomery
only at boundaries (after sampling/unpacking, before Decompose/Norm/packing).
Includes lazy MDS reduction (accumulate in uint64, single reduction per row).

#### 5. Zero-Allocation Poseidon
Add scratch buffer to Poseidon struct, reuse across rounds.
Reduced allocations from ~7000 to ~110 per Sign.

#### 6. Optimized Add/Sub
Use uint32/int32 arithmetic instead of uint64. Since Q < 2^23, we have a+b < 2^24
which fits in uint32.

### Optimizations That Did NOT Work

#### 1. Solinas/Proth Reduction
Tried exploiting Q = 7·2^20 + 1 structure for faster reduction.
Montgomery multiplication empirically outperformed it on ARM64.

#### 2. ILP with 5 Parallel Accumulators in MDS
Theory: Break dependency chain by using 5 independent accumulators.
Reality: ARM64 out-of-order execution already handles this. No improvement.

#### 3. Precomputed Inv2*InvZeta Table for InvNTT
Added 1KB lookup table to combine two multiplications into one.
Made it slower due to cache effects. Reverted.

#### 4. Moving Loop-Invariant MulMont Outside Inner Loop
Compiler already optimizes this. No improvement.

#### 5. Bounds Check Elimination Hints
Tried various slice patterns to help Go eliminate bounds checks.
No measurable improvement - compiler is already smart.

#### 6. Unsafe Pointers to Eliminate Bounds Checks
Cast slices to fixed-size array pointers: `(*[35]uint32)(unsafe.Pointer(&s[0]))`.
No improvement. Go's bounds check elimination is good enough.

#### 7. SqrMont (Specialized Squaring)
Specialized `a*a` function for InvMont. Theory: one fewer register load.
No measurable improvement.

#### 8. Zero-Copy BatchInvMontTo
Write inverses directly to output buffer instead of in-place + copy.
The copy was only 0.6% of runtime. No measurable improvement.

### Profiling Breakdown (Final)

Sign operation (6.0ms):
- poseidonRound: 90% (MDS matrix + batch inversion)
- MulMont: 24% (called from BatchInvMont and InvMont)
- BatchInvMont: 17% flat, 41% cumulative
- MontReduce: 4%
- NTT/InvNTT: 5% combined
- SHA3/SHAKE: 1-2%

### What Would Help Further

1. **ARM64 NEON Assembly** - SIMD could parallelize MulMont operations
2. **Assembly MDS** - Vectorized matrix-vector product
3. **Different Algorithm** - But we're implementing a specific spec

### Architecture Notes

- Platform: linux/arm64
- Field: Z_Q where Q = 2^23 - 2^20 + 1 = 7340033 (23-bit prime)
- Montgomery R = 2^32
- Poseidon: T=35 (state width), Rate=24, RF=21 (full rounds)
- Polynomial degree: N=256

### Useful Commands

```bash
# Run benchmarks
go test ./pkg/dilithium/ -bench=. -benchmem

# CPU profile
go test ./pkg/dilithium/ -bench=BenchmarkSign -cpuprofile=cpu.prof
go tool pprof -top cpu.prof

# Memory profile
go test ./pkg/dilithium/ -bench=BenchmarkSign -memprofile=mem.prof
go tool pprof -top mem.prof

# View assembly for a function
go build -gcflags="-S" ./pkg/field/ 2>&1 | grep -A 50 '"".MulMont'
```
