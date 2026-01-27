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

| Operation | Go | Python | Speedup |
|-----------|-----|--------|---------|
| Gen (keygen) | 0.10 ms | 3.1 ms | 31x |
| Sign | 19.8 ms | 461 ms | 23x |
| Verify | 2.9 ms | 71.5 ms | 25x |

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
