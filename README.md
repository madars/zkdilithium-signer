# zkdilithium-signer

> **WARNING: VIBE CODED** - This implementation was primarily written by AI (Claude). The code has been tested against the reference Python implementation with 1000+ test vectors, but has not been audited. Use at your own risk.

A Go implementation of the STARK-friendly Dilithium signature scheme from Cloudflare's [zkDilithium](https://github.com/guruvamsi-policharla/zkdilithium) research project.

This package implements **only the signature scheme** (key generation, signing, verification), not the zero-knowledge proof system. The full zkDilithium system uses this signature scheme to enable ZK proofs of signature possession.

## How This Differs from Standard Dilithium

Standard [Dilithium](https://pq-crystals.org/dilithium/) (NIST FIPS 204) uses parameters optimized for general-purpose post-quantum signatures. The zkDilithium variant modifies Dilithium2 for compatibility with STARK proof systems:

| Parameter | Standard Dilithium2 | zkDilithium |
|-----------|---------------------|-------------|
| Prime modulus Q | 8380417 | 7340033 (2^23 - 2^20 + 1) |
| Hash function | SHA3/SHAKE | Poseidon |
| Public key | Compressed (t1) | Full (t) |
| Challenge sampling | SHAKE-based | Poseidon-based |

**Key differences explained:**

1. **Poseidon hash** - Replaces SHA3/SHAKE with Poseidon, which has efficient arithmetic circuit representations for STARKs.

2. **STARK-friendly prime** - Q = 7340033 enables efficient NTT in STARK proofs (vs 8380417 in standard Dilithium).

3. **No public key compression** - Full `t` vector instead of `t1` from Power2Round, simplifying the arithmetic circuit.

4. **Modified sampleInBall** - Adjusted for deterministic execution in proof systems.

## License

MIT License - see [LICENSE](LICENSE) for details.

Based on the Python reference implementation from [guruvamsi-policharla/zkdilithium](https://github.com/guruvamsi-policharla/zkdilithium) (MIT License, Copyright 2023 Cloudflare, Guru-Vamsi Policharla).

## Bug Fix from Upstream

This repository includes a fix for a bug in the upstream Python implementation's `Poly.norm()` function. The bug caused incorrect handling of negative integers from `decompose()`, which could produce signatures that pass signing but fail verification. See [UPSTREAM_BUG_REPORT.txt](UPSTREAM_BUG_REPORT.txt) for details.

## Project Structure

```
├── zkdilithium.py              # Reference Python implementation (with bug fix)
├── test_zkdilithium.py         # Test vectors with hardcoded expected values
├── benchmark_zkdilithium.py    # Performance benchmarks
├── generate_test_vectors.py    # Script to regenerate test vectors (deterministic)
├── generate_stress_vectors.py  # Generate 1000 test vectors (generates stress_vectors.json)
├── requirements.txt            # Python dependencies
├── LICENSE                     # MIT License
├── UPSTREAM_BUG_REPORT.txt     # Bug report for upstream
└── pkg/                        # Go implementation
    ├── field/                  # Modular arithmetic
    ├── poly/                   # Polynomial ring operations
    ├── ntt/                    # Number Theoretic Transform
    ├── hash/                   # Poseidon, Grain LFSR, XOF
    ├── encoding/               # Packing/unpacking
    ├── sampling/               # Uniform, bounded, and ball sampling
    └── dilithium/              # Key generation, signing, verification
```

## Performance

Benchmarks on Debian VM (MacBook M3 Pro host) with varied messages to capture
rejection sampling variance:

| Operation | Go | Go (optimized) | Python | vs Python | vs Go |
|-----------|-----|----------------|--------|-----------|-------|
| Gen | 0.10 ms | 0.077 ms | 3.1 ms | 40x | 1.3x |
| Sign | 19.8 ms | 3.41 ms | 461 ms | 135x | 5.8x |
| Verify | 2.9 ms | 0.58 ms | 71.5 ms | 123x | 5.0x |

*For comparison, pure Go Ed25519 (`go test -tags=purego`) achieves 0.020ms sign / 0.043ms verify.
zkDilithium is ~250x slower, partly due to the STARK-friendly Poseidon hash, and partly because
this implementation prioritizes correctness over performance (no assembly, limited optimization).
Go's Ed25519 has been refined over many years by expert cryptographers.*

### Optimizations

1. **Batch inversion for Poseidon S-box** - Uses Montgomery's trick to compute
   n inversions with only 1 inversion + 3(n-1) multiplications.

2. **Optimized modular inverse** - Uses a fixed addition chain for a^(Q-2)
   exploiting Q-2 = 0b110\_11111111111111111111. Reduces operations from ~43
   to 30 per Inv() call.

3. **Montgomery multiplication** - All NTT and Poseidon operations stay in
   Montgomery form. Convert to/from Montgomery only at boundaries.

4. **Lazy Montgomery reduction** - Skip conditional subtraction in multiplication
   chains, reducing operations by ~5%.

5. **Tree-based batch inversion** - Replaces sequential O(n) prefix products with
   O(log n) depth binary tree. Each layer's operations are independent, enabling
   instruction-level parallelism (~30% faster batch inversion, ~9% faster Sign).

6. **Optimized Add/Sub** - Uses uint32/int32 arithmetic instead of uint64,
   avoiding unnecessary promotion since Q < 2^23.

7. **MDS loop optimization** - Uses fixed-size array pointers to eliminate bounds
   checks, unrolls inner loop by 5 (35 = 7 × 5, benchmarked faster than 7-unroll on ARM64).

8. **Zero-allocation Poseidon** - Reusable scratch buffers reduce allocations
   from ~7000 to ~91 per Sign.

9. **Gen-specific optimizations** - Streaming XOF (reuse one rate-sized buffer instead of
   allocating 1344 bytes per polynomial), precompute s1Hat (NTT once instead of K times),
   and skip Montgomery form (use regular Mul for matrix-vector product). Results: 24%
   faster, 60% less memory.

10. **Branchless reduce** - Uses sign-bit mask instead of conditional branch for
    reducing values from <2Q to <Q. Avoids ~50% branch misprediction.

11. **Fused reduction in BatchInvMont** - Performs final reduction during backward
    pass instead of separate O(n) loop, improving cache locality.

12. **Lazy matrix-vector multiply** - Accumulates L=4 products in uint64 with single
    MontReduce per coefficient, instead of L MulMont + (L-1) Add with conditional subs.
    Also precomputes NTT(y) once instead of K×L=16 times in Sign.

13. **Conditional batch inversion dispatch + 4-pair ILP** - Scans for zeros before batch
    inversion; dispatches to faster NoZero path when none found (almost always). Added 4-pair
    unrolling in tree up-sweep/down-sweep for better instruction-level parallelism.
    Batch inversion 212ns → 175ns (~17%), Sign ~8% faster.

14. **MDS 3-accumulator ILP** - Fully unrolls 35-element MDS inner loop with 3 independent
    accumulator chains (s01, s23, s4) instead of serial MADD chain. Enables instruction-level
    parallelism across the 2+2+1 grouping pattern. MDS ~4% faster, Sign ~18% faster.

15. **InvNTT loop-invariant hoist** - Hoists `inv2zMont := MulMont(Inv2Mont, z)` out of the
    inner butterfly loop in `InvNTT`, removing redundant work per coefficient.

16. **Fixed-size Poseidon batch inversion path (n=35)** - Adds specialized no-zero tree inversion
    for Poseidon state width and removes an internal copy in that path. Improves Sign/Verify by
    another ~2-3% on top of previous tree-batch inversion improvements.

See [NOTES.md](NOTES.md) for detailed optimization journey and profiling analysis.

Run benchmarks:

```bash
# Go benchmarks
go test ./pkg/dilithium/ -bench=.

# Python benchmarks
source .venv/bin/activate
python benchmark_zkdilithium.py
```

## Go Usage

```go
import "zkdilithium-signer/pkg/dilithium"

// Generate keypair from 32-byte seed
seed := make([]byte, 32)
rand.Read(seed)
pk, sk := dilithium.Gen(seed)

// Sign a message
msg := []byte("hello world")
sig := dilithium.Sign(sk, msg)

// Verify signature
valid := dilithium.Verify(pk, msg, sig)
```

### Run Go Tests

```bash
# Run all tests
go test ./pkg/... -v

# Run only the gold standard signature test
go test ./pkg/dilithium/... -v -run TestSignFullSignature

# Run stress tests (requires stress_vectors.json)
go test ./pkg/dilithium/... -v -run TestStressVectors
```

## Python Setup

### Prerequisites (Debian/Ubuntu)

```bash
apt install python3 python3-venv golang
```

- Python 3.10+ (for SHAKE digest length parameter)
- Go 1.21+ (for the port)

### Create Virtual Environment

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

### Run Python Tests

```bash
# Run the test suite (71 tests)
pytest test_zkdilithium.py -v

# Run only the signature tests
pytest test_zkdilithium.py::TestSignature -v
```

## Test Vector Workflow

The tests use **hardcoded expected values** generated from the reference implementation. This allows byte-for-byte verification that Go matches Python.

```bash
# Generate stress test vectors (1000 random signatures)
source .venv/bin/activate
python generate_stress_vectors.py  # Creates stress_vectors.json
go test ./pkg/dilithium/... -v -run TestStressVectors
```

### The Gold Standard Test

The most important test verifies byte-for-byte signature match:

```python
pk, sk = Gen(bytes(32))       # Deterministic key generation
sig = Sign(sk, b'test')       # Deterministic signing
assert sig.hex() == EXPECTED  # 2340-byte signature must match exactly
```

If the Go implementation passes this test, all internal components are working correctly.

## References

- [Post-Quantum Privacy Pass via Post-Quantum Anonymous Credentials](https://eprint.iacr.org/2023/414) - The zkDilithium paper (Policharla, Westerbaan, Faz-Hernández, Wood - Cloudflare)
- [zkDilithium Python implementation](https://github.com/guruvamsi-policharla/zkdilithium) - Reference implementation
- [Dilithium specification](https://pq-crystals.org/dilithium/) - CRYSTALS-Dilithium (NIST FIPS 204)
- [Poseidon hash](https://eprint.iacr.org/2019/458) - STARK-friendly hash function

## Algorithm Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| Q | 7340033 | Prime modulus (2^23 - 2^20 + 1) |
| N | 256 | Polynomial degree |
| K, L | 4 | Matrix dimensions |
| η | 2 | Secret key coefficient bound |
| γ1 | 2^17 | Signature coefficient bound |
| γ2 | 65536 | Decomposition parameter |
| τ | 40 | Challenge weight |
| β | 80 | Norm bound (τ × η) |
