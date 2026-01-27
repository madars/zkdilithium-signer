#!/usr/bin/env python3
"""
Benchmarks for zkDilithium signature scheme.

Usage:
    source .venv/bin/activate
    python benchmark_zkdilithium.py
"""

import time
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
    print()

    # Fixed seed for reproducibility (matches Go benchmarks)
    seed = bytes.fromhex("80fbd4ab1316a32533956567386edf851a154a714a4d2aa77dbc85b176cb88d4")

    # Varied messages to capture rejection sampling variance
    msgs = [f"benchmark message {i}".encode() for i in range(100)]

    # Pre-generate keys for sign/verify benchmarks
    pk, sk = Gen(seed)
    sig = Sign(sk, msgs[0])

    # Benchmark Gen
    def bench_gen():
        Gen(seed)

    benchmark("Gen (keygen)", bench_gen, iterations=100)

    # Benchmark Sign (with varied messages to capture rejection sampling variance)
    sign_iter = [0]
    def bench_sign():
        Sign(sk, msgs[sign_iter[0] % len(msgs)])
        sign_iter[0] += 1

    benchmark("Sign", bench_sign, iterations=100)

    # Benchmark Verify
    def bench_verify():
        Verify(pk, msgs[0], sig)

    benchmark("Verify", bench_verify, iterations=100)

    # Benchmark Sign+Verify (with varied messages)
    sv_iter = [0]
    def bench_sign_verify():
        msg = msgs[sv_iter[0] % len(msgs)]
        s = Sign(sk, msg)
        Verify(pk, msg, s)
        sv_iter[0] += 1

    benchmark("Sign+Verify", bench_sign_verify, iterations=100)

    print()
    print("Note: Python benchmarks run with CPython. PyPy may be faster")
    print("but is not compatible with numpy C extensions.")


if __name__ == "__main__":
    main()
