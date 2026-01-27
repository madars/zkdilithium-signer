// Package hash provides hash functions for zkDilithium.
package hash

import (
	"golang.org/x/crypto/sha3"
)

// XOF128 returns SHAKE-128 output for seed||nonce.
func XOF128(seed []byte, nonce uint16) []byte {
	h := sha3.NewShake128()
	h.Write(seed)
	h.Write([]byte{byte(nonce & 0xFF), byte(nonce >> 8)})
	out := make([]byte, 1344) // Same as Python
	h.Read(out)
	return out
}

// XOF256 returns SHAKE-256 output for seed||nonce.
func XOF256(seed []byte, nonce uint16) []byte {
	h := sha3.NewShake256()
	h.Write(seed)
	h.Write([]byte{byte(nonce & 0xFF), byte(nonce >> 8)})
	out := make([]byte, 2*136) // Same as Python
	h.Read(out)
	return out
}

// H returns SHAKE-256 output of specified length.
func H(msg []byte, length int) []byte {
	h := sha3.NewShake256()
	h.Write(msg)
	out := make([]byte, length)
	h.Read(out)
	return out
}
