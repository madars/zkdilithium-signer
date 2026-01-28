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
- Sign: 4.2ms (4.7x faster than baseline 19.8ms)
- Verify: 0.66ms (4.4x faster than baseline 2.9ms)
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

#### 7. MDS Loop Optimization
- Use fixed-size array pointers `(*[35]uint32)` to eliminate bounds checks
- Unroll inner loop by 7 (35 = 5 × 7) to reduce loop overhead
- Use local temporaries (t0-t6) instead of accumulating directly
- Results: MDS time reduced ~16%, overall Sign ~7% faster

#### 8. Lazy Montgomery Reduction
- Skip conditional subtraction in MulMont for internal chains
- mulMontLazy outputs values < 2Q instead of < Q
- Safe because: for Q=7340033, R=2^32, inputs < 2Q → output < 4Q²/R + Q < 2Q
- Applied to InvMont (30 muls) and BatchInvMont (102 muls per batch)
- Single reduce() at chain end instead of per-operation
- Results: Sign 5.5ms → 5.25ms (~5%), Verify 0.85ms → 0.81ms (~5%)

#### 9. BatchInvMontParallel (Pair Processing for ILP)
- Process pairs of elements in forward/backward passes for instruction-level parallelism
- Branchless zero handling: `nonZeroMask = (x | -x) >> 63` (1 if nonzero, 0 if zero)
- Uniform operations: `safe = x*nzm + oneM*(1-nzm)` selects x or 1_M without branching
- Interleave multiplications: start two MULs before completing Montgomery reductions
- Results: BatchInvMont 342ns → 306ns (~10%), Sign ~5.2ms → ~5.0ms (~4%), Verify ~810μs → ~780μs (~4%)

#### 10. Gen Optimizations (Key Generation)
Gen doesn't use Poseidon, so Sign/Verify optimizations don't apply. Gen's bottleneck is SHA3/SHAKE
for sampling (40%+ of runtime). Applied three optimizations:

**Streaming XOF** - Instead of allocating 1344-byte buffers per polynomial, use a single
reusable 168-byte buffer (one SHAKE128 rate). The buffer is refilled as needed during
rejection sampling. Memory reduced from 39KB to 15.5KB per Gen (-60%). Note: Keccak
permutation count is similar since Go's sha3 library buffers internally.

**Precompute s1Hat** - The inner loop computed NTT(s1[j]) for each row of A (K=4 times).
Since s1 doesn't change, precompute s1Hat = NTT(s1) once and reuse. Reduced NTT calls from
16 to 4, saving ~17% of Gen time.

**Skip Montgomery for Gen** - Gen only needs A*s1 multiplication, not the full Montgomery
chain used in Sign. Changed to use `ntt.MulNTT` (regular Mul) instead of `poly.MulNTT`
(MulMont), eliminating ToMont/FromMont conversions on A, s1, s2, and t. Saved ~5%.

Results: Gen 100μs → 76μs (**24% faster**), memory 39KB → 15.5KB (**60% smaller**).

#### 11. Branchless Reduce and bits.Reverse8
- `reduce()` now uses sign-bit mask: `b + (Q & uint32(int32(b)>>31))` instead of
  `if a >= Q { return a - Q }`. Avoids ~50% branch misprediction for uniform inputs.
- `Brv()` uses `bits.Reverse8()` which compiles to RBIT on ARM64.
- Minimal measurable impact but cleaner code.

#### 12. Fused Reduction in BatchInvMont
- Previous: backward pass used `mulMontLazy`, then separate O(n) loop to reduce all outputs.
- Now: backward pass uses `MulMont` (strict) for `xs[i]`, fusing reduction into the pass.
- Saves O(n) memory reads/writes, improves cache locality.
- Results: ~5% faster BatchInvMont.

#### 13. Lazy Matrix-Vector Multiply
- `w = A * y` in Sign: accumulate L=4 products in uint64, single MontReduce per coefficient.
- Previous: L `MulMont` + (L-1) `Add` = 7 conditional subtractions per coefficient.
- Now: L multiplies + 1 MontReduce = 1 conditional subtraction per coefficient.
- Also fixed redundant NTT: `NTT(y[j])` was computed K×L=16 times, now L=4 times.
- Results: Sign ~1.6% faster.

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

#### 9. BatchInvMontNonZero (removing zero checks)
Theory: Poseidon state is never zero after adding round constants.
Reality: While rare (~1/Q probability per element), zeros DO occur in Poseidon
state during real signing over many test vectors. The optimization broke
correctness on stress tests.

#### 10. Precomputed Full 35×35 MDS Matrix
Store `MDS[i][j]` instead of computing `PosInvMont[i+j]`.
Made it 8% slower due to cache effects (4.9KB vs 276 bytes).

#### 11. PGO (Profile-Guided Optimization)
Tried Go 1.21+ PGO with a CPU profile.
Made Sign ~7% slower, possibly due to code layout changes hurting cache.

#### 12. Platelet's Fast Fixed-Multiplier Technique for NTT/MDS
Technique by platelet: https://codeforces.com/blog/entry/111566
For `a * k mod m` with fixed k,m: precompute `p = ceil(k * 2^64 / m)`, then
`result = hi((a * p mod 2^64) * m)`. Extends to sums: `(∑ai*bi) mod m = hi((∑ai*pi) * m)`.
Applied to NTT (zeta multiplications) and MDS (matrix entries are fixed).
Result: ~2% improvement - not worth the complexity. The technique replaces MontReduce
with a simpler hi-mul, but MontReduce is already efficient (2 muls vs 1 mul).
The real bottleneck is batch inversion (46% of runtime), not multiplication/reduction.

#### 13. Pure Branchless BatchInvMont (without pair processing)
Converted BatchInvMont to use branchless zero detection without pair processing.
Result: ~1.5% SLOWER (345ns vs 340ns). The branch predictor wins for rare zeros
(~1/Q probability). Branchless only pays off when combined with pair processing
that enables instruction-level parallelism.

#### 14. NEON SIMD Assembly for BatchInvMont
Attempted hand-written ARM64 NEON assembly for vectorized batch inversion.
Failed due to Go's calling convention complexity (regabi). Go generates wrappers
that convert between register ABI and stack-based calling, making it difficult
to write correct assembly that interoperates with Go code. Would need CGO or
a pure assembly implementation to avoid these issues.

#### 15. MDS Row Parallelization (2-row and 5-row)
Tried processing multiple MDS rows simultaneously for better ILP:
- 2-row parallel: No improvement (~304ns both). CPU out-of-order execution
  already parallelizes the 7 independent multiplications within each row.
- 5-row parallel: 26% slower (382ns vs 304ns). Too much register pressure
  with 5 accumulators and 5 inverse coefficient pointers.
The original single-row with 7-way unrolling is optimal.

#### 16. Hoisting Q Constant in Round Constant Loop
Assembly inspection shows Q = 7340033 is reloaded each iteration (2 instructions
per iteration × 35 iterations × 21 rounds = 1470 extra instructions per perm).
Estimated impact: ~0.1% of Sign time. Not worth hand-writing assembly.

### Profiling Breakdown (Final)

Sign operation (~4.2ms):
- poseidonRound: 42% (main hotspot)
  - BatchInvMontTree: 16% (S-box inversion, tree-based)
  - MDS matrix: ~23% (7-unrolled MADD chains)
  - Round constants: ~3%
- mulMontLazy: 20% (Montgomery multiplication)
- MontReduce: 3.5% (final reduction in MDS)
- NTT: 3%, InvNTT: 2%
- SHA3/SHAKE: ~2%

All hot paths already use branchless operations:
- Add/Sub: compiler generates CSEL (conditional select)
- MDS: 7-way unrolled, efficient MUL/MADD chains
- NTT: uses Add/Sub which compile to CSEL
- reduce: branchless sign-bit mask

### What Would Help Further

At this point, pure Go optimizations are exhausted. The 4x speedup from baseline
(19.8ms → 5.0ms) captures most available gains. Further improvement requires:

1. **Hand-written SIMD Assembly** - NEON could parallelize 4 MulMont operations
   simultaneously, but Go's calling conventions make this difficult without CGO.
2. **Algorithmic Changes** - Different Poseidon parameters (smaller T, fewer rounds)
   or a different hash function entirely. But we're implementing a specific spec.
3. **Specialized Hardware** - FPGA/ASIC for field arithmetic.

The codebase is now well-optimized:
- All hot paths use branchless operations (CSEL, sign-bit masks)
- Montgomery multiplication with lazy reduction
- Tree-based batch inversion with O(log n) depth
- Lazy matrix-vector multiply with fused accumulation
- MDS matrix with 7-way unrolled MADD chains
- Zero-allocation Poseidon with reusable scratch buffers

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
