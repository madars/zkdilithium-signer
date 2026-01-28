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

// StreamingXOF128 provides incremental SHAKE-128 output.
type StreamingXOF128 struct {
	h   sha3.ShakeHash
	buf [168]byte // SHAKE128 rate
	pos int
	end int
}

// NewStreamingXOF128 creates a streaming XOF for seed||nonce.
func NewStreamingXOF128(seed []byte, nonce uint16) *StreamingXOF128 {
	h := sha3.NewShake128()
	h.Write(seed)
	h.Write([]byte{byte(nonce & 0xFF), byte(nonce >> 8)})
	return &StreamingXOF128{h: h}
}

// Read3 returns the next 3 bytes from the XOF.
func (x *StreamingXOF128) Read3() (b0, b1, b2 byte) {
	if x.pos+3 > x.end {
		// Copy leftover bytes to beginning
		leftover := x.end - x.pos
		if leftover > 0 {
			copy(x.buf[:leftover], x.buf[x.pos:x.end])
		}
		// Refill rest of buffer
		n, _ := x.h.Read(x.buf[leftover:])
		x.pos = 0
		x.end = leftover + n
	}
	b0, b1, b2 = x.buf[x.pos], x.buf[x.pos+1], x.buf[x.pos+2]
	x.pos += 3
	return
}

// Reset reinitializes the XOF for a new seed||nonce.
func (x *StreamingXOF128) Reset(seed []byte, nonce uint16) {
	x.h.Reset()
	x.h.Write(seed)
	x.h.Write([]byte{byte(nonce & 0xFF), byte(nonce >> 8)})
	x.pos = 0
	x.end = 0
}

// NewStreamingXOF128Reusable creates a reusable streaming XOF.
func NewStreamingXOF128Reusable() *StreamingXOF128 {
	return &StreamingXOF128{h: sha3.NewShake128()}
}

// SeedClonableXOF128 supports cloning state after seed absorption.
// This avoids re-hashing the seed for each nonce.
type SeedClonableXOF128 struct {
	seedState sha3.ShakeHash // State after absorbing seed (before nonce)
	h         sha3.ShakeHash // Current working hash
	buf       [168]byte
	pos       int
	end       int
}

// clonable interface for sha3.ShakeHash
type clonable interface {
	Clone() sha3.ShakeHash
}

// NewSeedClonableXOF128 creates an XOF with seed pre-absorbed.
func NewSeedClonableXOF128(seed []byte) *SeedClonableXOF128 {
	h := sha3.NewShake128()
	h.Write(seed)
	return &SeedClonableXOF128{
		seedState: h.(clonable).Clone(),
		h:         h,
	}
}

// SetNonce sets the nonce and prepares for reading.
// Efficiently restores from the seed-absorbed state.
func (x *SeedClonableXOF128) SetNonce(nonce uint16) {
	x.h = x.seedState.(clonable).Clone()
	x.h.Write([]byte{byte(nonce & 0xFF), byte(nonce >> 8)})
	x.pos = 0
	x.end = 0
}

// Read3 returns the next 3 bytes from the XOF.
func (x *SeedClonableXOF128) Read3() (b0, b1, b2 byte) {
	if x.pos+3 > x.end {
		// Copy leftover bytes to beginning
		leftover := x.end - x.pos
		if leftover > 0 {
			copy(x.buf[:leftover], x.buf[x.pos:x.end])
		}
		// Refill rest of buffer
		n, _ := x.h.Read(x.buf[leftover:])
		x.pos = 0
		x.end = leftover + n
	}
	b0, b1, b2 = x.buf[x.pos], x.buf[x.pos+1], x.buf[x.pos+2]
	x.pos += 3
	return
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

// StreamingXOF256 provides incremental SHAKE-256 output.
type StreamingXOF256 struct {
	h   sha3.ShakeHash
	buf [136]byte // SHAKE256 rate
	pos int
	end int
}

// NewStreamingXOF256Reusable creates a reusable streaming XOF256.
func NewStreamingXOF256Reusable() *StreamingXOF256 {
	return &StreamingXOF256{h: sha3.NewShake256()}
}

// Reset reinitializes the XOF for a new seed||nonce.
func (x *StreamingXOF256) Reset(seed []byte, nonce uint16) {
	x.h.Reset()
	x.h.Write(seed)
	x.h.Write([]byte{byte(nonce & 0xFF), byte(nonce >> 8)})
	x.pos = 0
	x.end = 0
}

// Read3 returns the next 3 bytes from the XOF.
func (x *StreamingXOF256) Read3() (b0, b1, b2 byte) {
	if x.pos+3 > x.end {
		// Copy leftover bytes to beginning
		leftover := x.end - x.pos
		if leftover > 0 {
			copy(x.buf[:leftover], x.buf[x.pos:x.end])
		}
		// Refill rest of buffer
		n, _ := x.h.Read(x.buf[leftover:])
		x.pos = 0
		x.end = leftover + n
	}
	b0, b1, b2 = x.buf[x.pos], x.buf[x.pos+1], x.buf[x.pos+2]
	x.pos += 3
	return
}

// H returns SHAKE-256 output of specified length.
func H(msg []byte, length int) []byte {
	h := sha3.NewShake256()
	h.Write(msg)
	out := make([]byte, length)
	h.Read(out)
	return out
}
