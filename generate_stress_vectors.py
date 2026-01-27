#!/usr/bin/env python3
"""Generate stress test vectors for zkDilithium Go implementation.

Generates 1000 signature test cases with random seeds and messages,
outputting JSON for Go to verify byte-for-byte compatibility.
"""

import json
import random
import sys
import zkdilithium as zk

def main():
    random.seed(20240102)  # Deterministic for reproducibility

    vectors = []

    # Suppress the retry messages from zkdilithium
    old_stdout = sys.stdout
    sys.stdout = open('/dev/null', 'w')

    for i in range(1000):
        # Random 32-byte seed
        seed = bytes(random.randint(0, 255) for _ in range(32))

        # Random message (1-100 bytes)
        msg_len = random.randint(1, 100)
        msg = bytes(random.randint(0, 255) for _ in range(msg_len))

        # Generate keys and sign
        pk, sk = zk.Gen(seed)
        sig = zk.Sign(sk, msg)

        # Verify signature is valid
        assert zk.Verify(pk, msg, sig), f"Verification failed for vector {i}"

        vectors.append({
            "seed": seed.hex(),
            "msg": msg.hex(),
            "pk": pk.hex(),
            "sk": sk.hex(),
            "sig": sig.hex(),
        })

        if (i + 1) % 100 == 0:
            sys.stdout = old_stdout
            print(f"Generated {i + 1}/1000 vectors", flush=True)
            sys.stdout = open('/dev/null', 'w')

    sys.stdout = old_stdout

    with open("stress_vectors.json", "w") as f:
        json.dump(vectors, f, indent=2)

    print(f"Generated {len(vectors)} test vectors to stress_vectors.json")

if __name__ == "__main__":
    main()
