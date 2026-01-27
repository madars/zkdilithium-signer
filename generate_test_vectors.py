#!/usr/bin/env python3
"""
Generate test vectors from zkdilithium.py

This script exercises the reference Python implementation and outputs
JSON test vectors. These vectors are then hardcoded into test_zkdilithium.py
for use in both Python and Go testing.

Usage:
    python generate_test_vectors.py > test_vectors.json

The output can be used to:
1. Verify the Python tests match the implementation
2. Port tests to Go by copying the expected values
3. Regenerate vectors if the implementation changes

IMPORTANT: This script must be deterministic. All randomness is seeded.
"""

import random
random.seed(20240101)  # Fixed seed for reproducibility

import json
from zkdilithium import (
    Q, N, ZETA, INVZETA, INV2, ZETAS, INVZETAS,
    GAMMA1, GAMMA2, ETA, K, L, TAU, BETA,
    POS_T, POS_RATE, POS_RF, POS_RCS, POS_INV,
    inv, brv, decompose,
    packFes, unpackFes,
    bytesToFes,
    XOF128, XOF256, H,
    poseidon_perm,
    sampleUniform, sampleLeqEta, sampleInBall,
    Cubic, Sextic, Poly, Vec, Matrix, Poseidon, Grain,
    Gen, Sign, Verify,
)


def generate_vectors():
    vectors = {}

    # === Constants ===
    vectors['constants'] = {
        'Q': Q,
        'N': N,
        'ZETA': ZETA,
        'INVZETA': INVZETA,
        'INV2': INV2,
        'GAMMA1': GAMMA1,
        'GAMMA2': GAMMA2,
        'ETA': ETA,
        'K': K,
        'L': L,
        'TAU': TAU,
        'BETA': BETA,
        'POS_T': POS_T,
        'POS_RATE': POS_RATE,
        'POS_RF': POS_RF,
    }

    # === First 16 ZETAS and INVZETAS ===
    vectors['zetas_first16'] = ZETAS[:16]
    vectors['invzetas_first16'] = INVZETAS[:16]

    # === Modular inverse ===
    vectors['inv'] = [
        {'input': 1, 'output': inv(1)},
        {'input': 2, 'output': inv(2)},
        {'input': 3, 'output': inv(3)},
        {'input': 1000, 'output': inv(1000)},
        {'input': Q - 1, 'output': inv(Q - 1)},
        {'input': 123456, 'output': inv(123456)},
    ]

    # === Bit reversal ===
    vectors['brv'] = [
        {'input': i, 'output': brv(i)}
        for i in [0, 1, 2, 127, 128, 255, 0b10101010]
    ]

    # === XOF128 ===
    stream = XOF128(b'\x00' * 32, 0)
    vectors['xof128'] = {
        'seed_hex': '00' * 32,
        'nonce': 0,
        'output_hex': stream.read(32).hex(),
    }

    stream2 = XOF128(b'\xab\xcd' + b'\x00' * 30, 42)
    vectors['xof128_2'] = {
        'seed_hex': 'abcd' + '00' * 30,
        'nonce': 42,
        'output_hex': stream2.read(32).hex(),
    }

    # === XOF256 ===
    stream = XOF256(b'\x00' * 64, 0)
    vectors['xof256'] = {
        'seed_hex': '00' * 64,
        'nonce': 0,
        'output_hex': stream.read(32).hex(),
    }

    # === H (SHAKE256) ===
    vectors['h_shake256'] = {
        'input_hex': b'test'.hex(),
        'length': 32,
        'output_hex': H(b'test', 32).hex(),
    }

    # === bytesToFes ===
    vectors['bytesToFes'] = [
        {'input_hex': bytes([0, 0]).hex(), 'output': bytesToFes(bytes([0, 0]))},
        {'input_hex': bytes([5]).hex(), 'output': bytesToFes(bytes([5]))},
        {'input_hex': bytes([0xff, 0xff]).hex(), 'output': bytesToFes(bytes([0xff, 0xff]))},
        {'input_hex': b'hello'.hex(), 'output': bytesToFes(b'hello')},
    ]

    # === packFes / unpackFes ===
    fes = [0, 1, 100, 1000, Q - 1, Q // 2]
    vectors['packFes'] = {
        'input': fes,
        'output_hex': packFes(fes).hex(),
    }

    # === decompose ===
    vectors['decompose'] = [
        {'input': r, 'r0': decompose(r)[0], 'r1': decompose(r)[1]}
        for r in [0, 1, GAMMA2, 2 * GAMMA2, Q - 1, Q - GAMMA2, Q // 2, 12345, Q - 12345]
    ]

    # === Cubic arithmetic ===
    a = Cubic(100, 200, 300)
    b = Cubic(400, 500, 600)
    ab = a * b
    vectors['cubic_mul'] = {
        'a': [100, 200, 300],
        'b': [400, 500, 600],
        'a_times_b': [ab.a0, ab.a1, ab.a2],
    }
    a_plus_b = a + b
    vectors['cubic_add'] = {
        'a': [100, 200, 300],
        'b': [400, 500, 600],
        'a_plus_b': [a_plus_b.a0, a_plus_b.a1, a_plus_b.a2],
    }

    # === Poly NTT ===
    p = Poly([1] + [0] * 255)
    p_ntt = p.NTT()
    vectors['ntt_of_one'] = {
        'input': [1] + [0] * 255,
        'output_first16': list(p_ntt.cs[:16]),
        'output_last16': list(p_ntt.cs[-16:]),
    }

    p2 = Poly(range(256))
    p2_ntt = p2.NTT()
    vectors['ntt_of_range'] = {
        'input_first16': list(range(16)),
        'output_first16': list(p2_ntt.cs[:16]),
        'output_last16': list(p2_ntt.cs[-16:]),
    }

    # === Poly schoolbook mul ===
    a = Poly(list(range(256)))
    b = Poly(list(range(256, 512)))
    q, r = a.SchoolbookMul(b)
    vectors['schoolbook_mul'] = {
        'a_first8': list(range(8)),
        'b_first8': list(range(256, 264)),
        'r_first16': list(r.cs[:16]),
        'r_last16': list(r.cs[-16:]),
        'q_first16': list(q.cs[:16]),
    }

    # === Poseidon permutation ===
    state = [i for i in range(POS_T)]
    poseidon_perm(state)
    vectors['poseidon_perm'] = {
        'input': list(range(POS_T)),
        'output': state,
    }

    # === Poseidon sponge ===
    h = Poseidon([1, 2, 3])
    out = h.read(12)
    vectors['poseidon_sponge'] = {
        'input': [1, 2, 3],
        'output': out,
    }

    # === Grain LFSR (first few field elements) ===
    g = Grain()
    grain_fes = [g.readFe() for _ in range(10)]
    vectors['grain_first10_fes'] = grain_fes

    # === POS_RCS first 16 ===
    vectors['pos_rcs_first16'] = POS_RCS[:16]

    # === sampleUniform ===
    p = sampleUniform(XOF128(b'\x00' * 32, 0))
    vectors['sampleUniform'] = {
        'seed_hex': '00' * 32,
        'nonce': 0,
        'output_first16': list(p.cs[:16]),
        'output_last16': list(p.cs[-16:]),
    }

    # === sampleLeqEta ===
    p = sampleLeqEta(XOF256(b'\x00' * 64, 0))
    vectors['sampleLeqEta'] = {
        'seed_hex': '00' * 64,
        'nonce': 0,
        'output_first16': list(p.cs[:16]),
    }

    # === sampleInBall ===
    for seed in range(100):
        h = Poseidon([2] + [seed] + [0] * 11)
        c = sampleInBall(h)
        if c is not None:
            nonzero_positions = [i for i, x in enumerate(c.cs) if x != 0]
            nonzero_values = [c.cs[i] for i in nonzero_positions]
            vectors['sampleInBall'] = {
                'poseidon_input': [2] + [seed] + [0] * 11,
                'nonzero_count': len(nonzero_positions),
                'nonzero_positions_first10': nonzero_positions[:10],
                'nonzero_values_first10': nonzero_values[:10],
            }
            break

    # === Key generation ===
    pk, sk = Gen(b'\x00' * 32)
    vectors['keygen'] = {
        'seed_hex': '00' * 32,
        'pk_len': len(pk),
        'sk_len': len(sk),
        'pk_first32_hex': pk[:32].hex(),
        'pk_bytes_32_to_64_hex': pk[32:64].hex(),
        'sk_first32_hex': sk[:32].hex(),
    }

    # === Sign and verify ===
    sig = Sign(sk, b'test')
    vectors['sign'] = {
        'message': 'test',
        'sig_len': len(sig),
        'sig_first32_hex': sig[:32].hex(),
        'verify_result': Verify(pk, b'test', sig),
    }

    return vectors


if __name__ == '__main__':
    vectors = generate_vectors()
    print(json.dumps(vectors, indent=2))
