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
- Sign: 3.4ms (5.8x faster than baseline 19.8ms)
- Verify: 0.54ms (5.4x faster than baseline 2.9ms)
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
- Unroll inner loop by 5 (35 = 7 × 5) - benchmarked faster than 7-unroll on ARM64
- Use local temporaries (t0-t4) instead of accumulating directly
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

#### 14. Conditional BatchInvMontTree Dispatch + 4-Pair ILP Unrolling
- Poseidon S-box uses batch inversion. Zero handling adds overhead (copy with substitution,
  conditional writeback) even though zeros are extremely rare (~1/Q ≈ 1/7M per element).
- New approach: scan for zeros first (O(n) comparisons), dispatch to fast path when no zeros
  found (almost always).
- Added 4-pair unrolling in up-sweep and down-sweep loops for better instruction-level
  parallelism. Loads 4 parent values and 8 child values, performs 8 independent mulMontLazy
  operations before storing results.
- Benchmarked unroll factors: no unroll (184ns), 2-pair (179ns), **4-pair (175ns)**.
- Results: BatchInvMontTree 212ns → 175ns (~17%), Sign 4.35ms → 4.0ms (~8%).

#### 15. MDS 3-Accumulator ILP with Full Unrolling
- Previous: 5-unrolled loop with serial MADD chain accumulating into single `acc`.
- Compiler generated: `MUL; MADD; MADD; MADD; MADD; ADD acc` - each MADD depends on previous.
- New approach: 3 independent accumulator chains (s01, s23, s4) with full unrolling of all 35 elements.
- Each group of 5: s01 += products[0,1], s23 += products[2,3], s4 += products[4].
- Final merge: `state[i] = MontReduce(s01 + s23 + s4)`.
- Key insight: 3 accumulators (not 5) hits the sweet spot - enough parallelism without register pressure.
- The 2+2+1 grouping matches ARM64's dual-issue capability for MUL/MADD.
- Results: MDS 294ns → 279ns (~5%), Sign 4.0ms → 3.4ms (~18%), Verify 650μs → 540μs (~17%).

#### 16. InvNTT Loop-Invariant Hoist (2026-02 revisit)
- In `InvNTT`, `inv2zMont := MulMont(Inv2Mont, z)` is invariant for each
  `(layer, offset)` block but was computed inside the inner butterfly loop.
- Hoisted `inv2zMont` outside the inner `j` loop.
- Controlled microbench A/B (same environment):
  - Before: `BenchmarkInvNTT` **~2025-2036ns**
  - After: `BenchmarkInvNTT` **~1738-1746ns**
  - Improvement: **~14%** on isolated InvNTT.
- End-to-end Sign/Verify impact is small (expected, since Poseidon dominates).

#### 17. Fixed-Size BatchInvMontTreeNoZeroILP4 Specialization (`n=35`)
- Poseidon always batch-inverts `PosT=35` elements. The generic
  `BatchInvMontTreeNoZeroILP4` still computed dynamic layer metadata and
  iterated over variable layer counts.
- Added fixed-size fast path:
  - `batchInvMontTreeNoZeroILP4_35` with pre-determined layers
    `35 -> 18 -> 9 -> 5 -> 3 -> 2 -> 1`.
  - `BatchInvMontTreeNoZeroILP4` now dispatches to this path when `len(xs)==35`.
- Controlled A/B (`-benchtime=2s`, same environment):
  - **Without** specialization: `Sign ~3.38-3.42ms`, `Verify ~558-561us`
  - **With** specialization: `Sign ~3.29-3.31ms`, `Verify ~542-546us`
  - Improvement: **~2.8% Sign**, **~2.9% Verify**.
- Microbench improvement for batch inversion itself is larger (~10% range),
  with smaller end-to-end impact due to other Poseidon costs.

#### 18. Copy Elision in `batchInvMontTreeNoZeroILP4_35`
- Initial fixed-size path still copied `xs -> scratch[:35]` before up-sweep.
- Revised fast path uses `xs` directly as level-0 storage and keeps only upper
  layers in scratch (`18+9+5+3+2+1 = 38` words).
- This removes per-call `memmove` and shrinks specialized assembly
  (`STEXT size 4064 -> 3984`, locals `0x48 -> 0x38`).
- Benchmarks (`-benchtime=2s`, same environment):
  - Before (fixed-size with copy): `Sign ~3.29-3.31ms`, `Verify ~542-546us`
  - After (copy-elided): `Sign ~3.20-3.25ms`, `Verify ~531-545us`
  - Additional gain: **~2-3% Sign** on top of #17.

### Optimizations That Did NOT Work

#### 1. Solinas/Proth Reduction
Tried exploiting Q = 7·2^20 + 1 structure for faster reduction.
Montgomery multiplication empirically outperformed it on ARM64.

#### 2. ILP with 5 Parallel Accumulators in MDS
Theory: Break dependency chain by using 5 independent accumulators.
Reality: ARM64 out-of-order execution already handles this. No improvement.
**Update:** 3 accumulators with full unrolling DOES work (~18% Sign improvement).
The key differences: (1) 3 accumulators vs 5 reduces register pressure, (2) full
unrolling eliminates loop overhead, (3) 2+2+1 grouping matches dual-issue better.

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

#### 17. Manual "Branchless" Arithmetic (Add/Sub/MontReduce)
Tried replacing `if` statements with mask-based branchless code:
```go
// Manual "branchless"
mask := uint32(int32(sum-Q) >> 31)
return sum - (Q & ^mask)
```
Result: **40% slower** for Add, **31% slower** for MontReduce.

The Go compiler on ARM64 already generates branchless code using CSEL:
```asm
ADD    R0, R1, R1     ; sum = a + b
MOVD   $Q, R2         ; load Q
SUB    R2, R1, R3     ; R3 = sum - Q
CMPW   R2, R1         ; compare
CSEL   HS, R3, R1, R0 ; branchless conditional select!
```
Manual mask computation adds extra instructions on top of what the compiler
already does optimally.

#### 18. Batched mulMontLazy2
Tried batching two Montgomery multiplications into one function call for ILP.
Result: **No improvement** (2.83ns vs 2.94ns). ARM64's out-of-order execution
already pipelines sequential mulMontLazy calls effectively.

#### 19. MDS Unroll Factor Experiments
Benchmarked different unroll factors for the 35-element MDS inner loop:
- **Unroll-5: 283ns** (winner, 35 = 7 × 5)
- Unroll-7: 299ns (previous)
- Fully unrolled: 327ns (register pressure)
- No unroll: 395ns

Unroll-5 is ~5% faster than unroll-7. The 7 outer iterations with 5 multiplies
each has better register allocation than 5 outer iterations with 7 multiplies.

#### 20. Go Assembly for mulMontLazy
Attempted hand-written ARM64 assembly for mulMontLazy (2 UMULL + 1 UMADDL + 1 LSR).
Result: **40x slower** (10ns vs 0.26ns). Go assembly for non-runtime packages uses
stack-based calling convention (abi0), not register ABI. Every call requires:
- Loading parameters from stack into registers
- Storing result back to stack
This overhead dominates the 4-instruction operation. Meanwhile, Go's inlined
mulMontLazy has zero call overhead.

The optimal ARM64 sequence (via `clang -O3`) is:
```asm
umull  x2, w0, w1       ; t = a * b (64-bit)
mul    w3, w2, w(QINV)  ; m = lo(t) * QINV_NEG
umaddl x2, w(Q), w3, x2 ; t += m * Q
lsr    x0, x2, #32      ; return hi(t)
```
Go generates equivalent code when mulMontLazy is inlined.

#### 21. Inlining Budget Analysis
Go's inliner has a cost budget of 80. Analyzed all field functions:

**Hot path functions that CAN inline (cost ≤ 80):**
- `mulMontLazy`: cost 32 ✓
- `MulMont`: cost 39 ✓
- `MontReduce`: cost 30 ✓
- `reduce`: cost 22 ✓
- `Add`: cost 16 ✓
- `Sub`: cost 19 ✓
- `ToMont/FromMont`: cost 44 ✓
- `MulMont2/MontReduce2`: cost 77/59 ✓

**Complex algorithms that CANNOT inline (shouldn't be split):**
- `InvMont`: cost 1150 (29 chained operations)
- `BatchInvMontTree`: cost 766 (complex tree algorithm)

**Experimental batched functions (not used in production):**
- `MontReduceLazy4`: cost 89 (only 9 over threshold)
- `MulMontLazy4`: cost 125
- `MulMont4`: cost 153

Splitting InvMont or BatchInvMontTree wouldn't help - they're inherently complex
and function call overhead would add cost. The batched 4-way functions showed
that call overhead negates ILP benefits when functions can't inline.

**Conclusion:** All hot-path functions already inline. The non-inlinable functions
are complex algorithms where splitting would hurt performance.

#### 22. Array-Pointer BCE Rewrite of `batchInvMontTreeNoZeroILP4_35`
Tried replacing slice layer views with fixed array pointers (`*[35]uint32`,
`*[18]uint32`, etc.) to force stronger bounds-check elimination.

Result: **Regression** in both microbench and end-to-end:
- BatchInv benchmarks moved back toward pre-specialization numbers.
- Sign/Verify regressed versus the slice-based specialized version.

Reverted.

#### 23. Dedicated `BatchInvMontTreeCond35` Entry Point
Tried adding a Poseidon-specific dispatch function and calling it from
`poseidonRound`, bypassing generic `BatchInvMontTreeCond -> NoZeroILP4` call chain.

Result: **No reliable additional gain** beyond the `n=35` specialization
(differences were within noise / inconsistent across runs).

Reverted to keep API surface minimal.

#### 24. Removing Pre-`reduce` Before Root `InvMont` (n=35 path)
Tried changing the specialized path root inversion from:
`InvMont(reduce(l6[0]))` to `InvMont(l6[0])`, attempting to carry lazy form longer.

Result: **Regression**:
- `BenchmarkBatchInvTreeCond35`: ~153ns -> ~158-160ns
- `BenchmarkBatchInvTreeNoZeroILP4_35`: ~144-148ns -> ~150-153ns

Likely cause: `InvMont` addition chain is tuned around canonical `<Q` inputs;
feeding `<2Q` representatives increases value distribution and hurts downstream
codegen/microarchitectural behavior enough to outweigh one removed `reduce`.

Reverted.

#### 25. Full Constant-Index Rewrite for `batchInvMontTreeNoZeroILP4_35`
Rewrote the specialized `n=35` batch inversion routine as explicit fixed-index
straight-line code (no loop-indexed slice accesses in the hot body), while keeping
the same algorithm and lazy/strict boundary decisions.

Assembly impact (`objdump`):
- Bounds-check/panic stubs in this symbol dropped from dozens of `panicIndex` callsites
  to only 2 upfront `panicSliceConvert` checks (for `(*[35]uint32)(xs)` and
  `(*[38]uint32)(scratch)` conversions).
- Symbol size dropped from ~1017 to ~841 assembly lines.
- `mulMontLazy` constants (`Q`, `QInv`) are loaded once and reused across long stretches.

Benchmarks after rewrite:
- `BenchmarkBatchInvTreeCond35`: ~148-151ns
- `BenchmarkBatchInvTreeNoZeroILP4_35`: ~147-148ns
- End-to-end (`go test ./pkg/dilithium -bench 'Benchmark(Gen|Sign|Verify)$' -benchtime=2s -count=3`):
  - Gen: ~0.073ms
  - Sign: ~3.10ms avg
  - Verify: ~0.51ms avg

Status: kept.

### Profiling Breakdown (Final)

Sign operation (~3.4ms):
- poseidonRound: 90% (main hotspot)
  - BatchInvMontTree: ~45% (S-box inversion, tree-based)
  - MDS matrix: ~45% (3-accumulator ILP, fully unrolled)
  - Round constants: ~3%
- NTT: 3%, InvNTT: 2%
- SHA3/SHAKE: ~2%

All hot paths already use branchless operations:
- Add/Sub: compiler generates CSEL (conditional select)
- MDS: 3-accumulator ILP with parallel MUL chains
- NTT: uses Add/Sub which compile to CSEL
- reduce: branchless sign-bit mask

### Primitive Microbench (2026-02, arm64)

`go test ./pkg/field -run '^$' -bench 'BenchmarkPrimitive(...)' -benchtime=2s -count=3`

| Primitive | Median ns/op | Relative to MulMont |
|-----------|--------------|---------------------|
| `reduce` | 1.203 | 0.64x |
| `Add` | 1.387 | 0.74x |
| `Sub` | 1.348 | 0.72x |
| `MontReduce` | 1.246 | 0.67x |
| `mulMontLazy` | 1.807 | 0.97x |
| `MulMont` | 1.872 | 1.00x |
| `reduce(mulMontLazy(...))` | 2.021 | 1.08x |
| `InvMont` | 45.04 | 24.1x |

Call-site implication:
- `mulMontLazy` alone is slightly cheaper than `MulMont`, but `reduce(mulMontLazy(...))`
  is more expensive than `MulMont`. So the gain comes from carrying lazy form across
  multiple operations and reducing only once at the boundary.

#### 26. Pure-Plain `n=35` ILP2 Rewrite (No Montgomery Correction in Timed Path)
For the plain-domain track, benchmarking must exclude Montgomery-domain correction
work (`*R^2`, `ToMont`/`FromMont`) because those conversions are not part of the
target design.

Changes:
- Added paired helpers in `field.go`:
  - `mulPlainLazy2(a0,b0,a1,b1)` (2-way lazy Barrett)
  - `mulPlainStrict2(a0,b0,a1,b1)` with inline two-lane branchless reduction
- Rewrote `batchInvTreeNoZeroILP4_35PlainLazy` to use paired operations through
  up-sweep/down-sweep/final writeback.

Checks:
- Functional tests pass (`TestBatchInvTreeNoZeroILP4_35PlainLazyMatches`, helper tests).
- Assembly for `batchInvTreeNoZeroILP4_35PlainLazy` still has only two upfront
  `panicSliceConvert` checks (no in-body bounds-check panics).

Benchmarks (arm64, `-benchtime=3s`):
- Before rewrite: `BenchmarkBatchInvTreeNoZeroILP4_35PlainLazy` ~149-150ns.
- After rewrite: `BenchmarkBatchInvTreeNoZeroILP4_35PlainLazy` median ~140.7ns.
- Relative improvement: ~6%.

This is now slightly faster than current mont-domain direct kernel in the same run.

Integration attempt (failed, reverted):
- Temporarily dispatched production `BatchInvMontTreeNoZeroILP4(n=35)` to
  `batchInvMontTreeNoZeroILP4_35LazyPlain`.
- Microbench direct kernels were roughly comparable, but end-to-end `Sign`
  regressed slightly in this environment (~3.20ms -> ~3.25ms).
- Reverted dispatch; keep this path as optimization track only for now.

#### 27. Corrected Folding for Fused 128-bit Reduction (Gemini Parallel Track)
For a 128-bit value `x = hi*2^64 + lo`, with `R64 = 2^64 mod Q = 3338324`:

- Naive one-fold truncate is UNSOUND:
  - `naive = (lo + hi*R64) mod 2^64`
  - Z3 finds counterexamples where `naive mod Q != x mod Q`.
- Correct carry-aware two-fold is sound over the tested batch-inversion-fused range:
  - `t = lo + hi*R64`
  - `corr = (t mod 2^64) + floor(t/2^64)*R64`
  - Then `corr mod Q == x mod Q` (Z3: `unsat` for mismatch in domain
    `x <= (2Q-1)^4`, i.e., 96-bit worst-case fused product bound).

Takeaway:
- If exploring fused-grandchild multiplication paths, use carry-aware two-fold
  folding before final `%Q`/Barrett reduction; do not use truncated one-fold.

### What Would Help Further

At this point, pure Go optimizations are largely exhausted. The 5.8x speedup from baseline
(19.8ms → 3.4ms) captures most available gains. Further improvement requires:

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
- MDS matrix with 3-accumulator ILP and full unrolling
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
