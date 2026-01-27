package dilithium

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

type StressVector struct {
	Seed string `json:"seed"`
	Msg  string `json:"msg"`
	Pk   string `json:"pk"`
	Sk   string `json:"sk"`
	Sig  string `json:"sig"`
}

// TestStressVectors verifies Go produces identical signatures to Python
// for 1000 random test cases.
func TestStressVectors(t *testing.T) {
	data, err := os.ReadFile("../../stress_vectors.json")
	if err != nil {
		t.Skip("stress_vectors.json not found, run generate_stress_vectors.py first")
	}

	var vectors []StressVector
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatalf("failed to parse stress_vectors.json: %v", err)
	}

	t.Logf("Testing %d stress vectors", len(vectors))

	for i, v := range vectors {
		seed, _ := hex.DecodeString(v.Seed)
		msg, _ := hex.DecodeString(v.Msg)
		expectedPk, _ := hex.DecodeString(v.Pk)
		expectedSk, _ := hex.DecodeString(v.Sk)
		expectedSig, _ := hex.DecodeString(v.Sig)

		pk, sk := Gen(seed)

		if string(pk) != string(expectedPk) {
			t.Errorf("vector %d: pk mismatch", i)
			continue
		}
		if string(sk) != string(expectedSk) {
			t.Errorf("vector %d: sk mismatch", i)
			continue
		}

		sig := Sign(sk, msg)
		if string(sig) != string(expectedSig) {
			t.Errorf("vector %d: sig mismatch at byte 0: got %02x, want %02x", i, sig[0], expectedSig[0])
			// Find first mismatch
			for j := range sig {
				if sig[j] != expectedSig[j] {
					t.Errorf("vector %d: first mismatch at byte %d: got %02x, want %02x", i, j, sig[j], expectedSig[j])
					break
				}
			}
			continue
		}

		// Verify signature
		if !Verify(pk, msg, sig) {
			t.Errorf("vector %d: signature verification failed", i)
		}

		if (i+1)%100 == 0 {
			t.Logf("Verified %d/%d vectors", i+1, len(vectors))
		}
	}
}
