// Package encoding provides serialization functions for zkDilithium.
package encoding

import "zkdilithium-signer/pkg/field"

// PackFes packs field elements into bytes (3 bytes per element, little-endian).
func PackFes(fes []uint32) []byte {
	result := make([]byte, len(fes)*3)
	for i, c := range fes {
		result[i*3] = byte(c & 0xFF)
		result[i*3+1] = byte((c >> 8) & 0xFF)
		result[i*3+2] = byte(c >> 16)
	}
	return result
}

// UnpackFes unpacks bytes into field elements.
func UnpackFes(bs []byte) []uint32 {
	n := len(bs) / 3
	result := make([]uint32, n)
	for i := 0; i < n; i++ {
		result[i] = (uint32(bs[i*3]) | (uint32(bs[i*3+1]) << 8) | (uint32(bs[i*3+2]) << 16)) % field.Q
	}
	return result
}

// BytesToFes converts bytes to field elements.
// Adds 1 to each byte and packs pairs into field elements.
// This distinguishes b'h' from b'h\0'.
func BytesToFes(bs []byte) []uint32 {
	// Add 1 to each byte (use uint32 to avoid overflow)
	modified := make([]uint32, len(bs))
	for i, b := range bs {
		modified[i] = uint32(b) + 1
	}

	// Pad to even length
	if len(modified)%2 == 1 {
		modified = append(modified, 0)
	}

	// Pack pairs
	result := make([]uint32, len(modified)/2)
	for i := 0; i < len(modified)/2; i++ {
		result[i] = modified[2*i] + 257*modified[2*i+1]
	}
	return result
}

// PackPoly packs a polynomial (256 coefficients) into bytes.
func PackPoly(cs *[field.N]uint32) []byte {
	return PackFes(cs[:])
}

// UnpackPoly unpacks bytes into a polynomial.
func UnpackPoly(bs []byte) [field.N]uint32 {
	var result [field.N]uint32
	fes := UnpackFes(bs)
	copy(result[:], fes)
	return result
}

// PackPolyLeqEta packs a polynomial with coefficients in [-Eta, Eta].
// Uses 3 bits per coefficient (8 coefficients per 3 bytes).
func PackPolyLeqEta(cs *[field.N]uint32) []byte {
	result := make([]byte, 96) // 256 * 3 / 8 = 96
	// Convert to [0, 2*Eta] range
	converted := make([]uint32, field.N)
	for i, c := range cs {
		// (Eta - c) mod Q
		converted[i] = field.Sub(field.Eta, c)
	}

	for i := 0; i < 256; i += 8 {
		j := i / 8 * 3
		result[j] = byte(converted[i]) | byte(converted[i+1]<<3) | byte((converted[i+2]<<6)&0xFF)
		result[j+1] = byte(converted[i+2]>>2) | byte(converted[i+3]<<1) | byte(converted[i+4]<<4) | byte((converted[i+5]<<7)&0xFF)
		result[j+2] = byte(converted[i+5]>>1) | byte(converted[i+6]<<2) | byte(converted[i+7]<<5)
	}
	return result
}

// UnpackPolyLeqEta unpacks a polynomial with coefficients in [-Eta, Eta].
func UnpackPolyLeqEta(bs []byte) [field.N]uint32 {
	var result [field.N]uint32
	idx := 0
	for i := 0; i < 96; i += 3 {
		result[idx] = uint32(bs[i] & 7)
		result[idx+1] = uint32((bs[i] >> 3) & 7)
		result[idx+2] = uint32((bs[i]>>6)|((bs[i+1]<<2)&7))
		result[idx+3] = uint32((bs[i+1] >> 1) & 7)
		result[idx+4] = uint32((bs[i+1] >> 4) & 7)
		result[idx+5] = uint32((bs[i+1]>>7)|((bs[i+2]<<1)&7))
		result[idx+6] = uint32((bs[i+2] >> 2) & 7)
		result[idx+7] = uint32((bs[i+2] >> 5) & 7)
		idx += 8
	}

	// Convert from [0, 2*Eta] back to [-Eta, Eta] mod Q
	for i := 0; i < field.N; i++ {
		result[i] = field.Mod(int64(field.Eta) - int64(result[i]))
	}
	return result
}

// PackPolyLeGamma1 packs a polynomial with coefficients in [-Gamma1+1, Gamma1].
// Uses 18 bits per coefficient (4 coefficients per 9 bytes).
func PackPolyLeGamma1(cs *[field.N]uint32) []byte {
	result := make([]byte, field.PolyLeGamma1Size) // 576 = 256 * 18 / 8

	for i := 0; i < 256; i += 4 {
		// Convert to [0, 2*Gamma1] range
		c0 := field.Sub(field.Gamma1, cs[i])
		c1 := field.Sub(field.Gamma1, cs[i+1])
		c2 := field.Sub(field.Gamma1, cs[i+2])
		c3 := field.Sub(field.Gamma1, cs[i+3])

		j := i / 4 * 9
		result[j] = byte(c0 & 0xFF)
		result[j+1] = byte((c0 >> 8) & 0xFF)
		result[j+2] = byte((c0 >> 16) | ((c1 << 2) & 0xFF))
		result[j+3] = byte((c1 >> 6) & 0xFF)
		result[j+4] = byte((c1 >> 14) | ((c2 << 4) & 0xFF))
		result[j+5] = byte((c2 >> 4) & 0xFF)
		result[j+6] = byte((c2 >> 12) | ((c3 << 6) & 0xFF))
		result[j+7] = byte((c3 >> 2) & 0xFF)
		result[j+8] = byte((c3 >> 10) & 0xFF)
	}
	return result
}

// UnpackPolyLeGamma1 unpacks a polynomial with coefficients in [-Gamma1+1, Gamma1].
func UnpackPolyLeGamma1(bs []byte) [field.N]uint32 {
	var result [field.N]uint32

	for i := 0; i < 64*9; i += 9 {
		c0 := uint32(bs[i]) | (uint32(bs[i+1]) << 8) | ((uint32(bs[i+2]) & 0x3) << 16)
		c1 := (uint32(bs[i+2]) >> 2) | (uint32(bs[i+3]) << 6) | ((uint32(bs[i+4]) & 0xF) << 14)
		c2 := (uint32(bs[i+4]) >> 4) | (uint32(bs[i+5]) << 4) | ((uint32(bs[i+6]) & 0x3F) << 12)
		c3 := (uint32(bs[i+6]) >> 6) | (uint32(bs[i+7]) << 2) | (uint32(bs[i+8]) << 10)

		idx := (i / 9) * 4
		result[idx] = field.Mod(int64(field.Gamma1) - int64(c0))
		result[idx+1] = field.Mod(int64(field.Gamma1) - int64(c1))
		result[idx+2] = field.Mod(int64(field.Gamma1) - int64(c2))
		result[idx+3] = field.Mod(int64(field.Gamma1) - int64(c3))
	}
	return result
}
