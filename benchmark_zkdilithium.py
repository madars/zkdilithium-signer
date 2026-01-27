#!/usr/bin/env python3
"""
Benchmarks for zkDilithium signature scheme.

Usage:
    source .venv/bin/activate
    python benchmark_zkdilithium.py
"""

import time
import os
from zkdilithium import Gen, Sign, Verify


def benchmark(name: str, func, iterations: int = 100):
    """Run a benchmark and print results."""
    # Warmup
    for _ in range(min(10, iterations // 10)):
        func()

    # Timed runs
    start = time.perf_counter()
    for _ in range(iterations):
        func()
    elapsed = time.perf_counter() - start

    avg_ms = (elapsed / iterations) * 1000
    ops_per_sec = iterations / elapsed
    print(f"{name}: {avg_ms:.3f} ms/op ({ops_per_sec:.1f} ops/sec, {iterations} iterations)")
    return avg_ms


def main():
    print("zkDilithium Benchmark")
    print("=" * 50)
    print(f"Message size: 64 bytes")
    print()

    # Fixed seed for reproducibility
    seed = os.urandom(32)
    msg = os.urandom(64)

    # Pre-generate keys for sign/verify benchmarks
    pk, sk = Gen(seed)
    sig = Sign(sk, msg)

    # Benchmark Gen
    def bench_gen():
        Gen(seed)

    benchmark("Gen (keygen)", bench_gen, iterations=100)

    # Benchmark Sign
    def bench_sign():
        Sign(sk, msg)

    benchmark("Sign", bench_sign, iterations=100)

    # Benchmark Verify
    def bench_verify():
        Verify(pk, msg, sig)

    benchmark("Verify", bench_verify, iterations=100)

    # Benchmark Sign+Verify
    def bench_sign_verify():
        s = Sign(sk, msg)
        Verify(pk, msg, s)

    benchmark("Sign+Verify", bench_sign_verify, iterations=100)

    print()
    print("Note: Python benchmarks run with CPython. PyPy may be faster")
    print("but is not compatible with numpy C extensions.")


if __name__ == "__main__":
    main()
