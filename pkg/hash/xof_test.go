package hash

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// Test XOF128 with known values from Python
func TestXOF128Zeros(t *testing.T) {
	seed := make([]byte, 32)
	got := XOF128(seed, 0)[:32]
	expected, _ := hex.DecodeString("49dfd9809bbc54014aabcc6a9a19f5ed48ad57d91902917201b689782ac6c75e")
	if !bytes.Equal(got, expected) {
		t.Errorf("XOF128(zeros, 0) = %x, want %x", got, expected)
	}
}

func TestXOF128WithData(t *testing.T) {
	// "abcd" + "00" * 30 = 32 bytes total
	seed, _ := hex.DecodeString("abcd000000000000000000000000000000000000000000000000000000000000")
	got := XOF128(seed, 42)[:32]
	expected, _ := hex.DecodeString("c284856075f7c4b04817d544b48d792c4793f2ce1215f04c812c58f9609617e1")
	if !bytes.Equal(got, expected) {
		t.Errorf("XOF128 with data = %x, want %x", got, expected)
	}
}

// Test XOF256 with known values from Python
func TestXOF256Zeros(t *testing.T) {
	seed := make([]byte, 64)
	got := XOF256(seed, 0)[:32]
	expected, _ := hex.DecodeString("4c838207f7a3088bf011c6d221a172bff9257c8f4b807ba9d4c851fd20263efb")
	if !bytes.Equal(got, expected) {
		t.Errorf("XOF256(zeros, 0) = %x, want %x", got, expected)
	}
}

// Test H (SHAKE-256) with known values from Python
func TestH(t *testing.T) {
	got := H([]byte("test"), 32)
	expected, _ := hex.DecodeString("b54ff7255705a71ee2925e4a3e30e41aed489a579d5595e0df13e32e1e4dd202")
	if !bytes.Equal(got, expected) {
		t.Errorf("H('test', 32) = %x, want %x", got, expected)
	}
}
