package dilithium

import (
	"crypto/rand"
	"testing"
)

// BenchmarkGen benchmarks key generation.
func BenchmarkGen(b *testing.B) {
	seed := make([]byte, 32)
	rand.Read(seed)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Gen(seed)
	}
}

// BenchmarkSign benchmarks signing with a 64-byte message.
func BenchmarkSign(b *testing.B) {
	seed := make([]byte, 32)
	rand.Read(seed)
	_, sk := Gen(seed)

	msg := make([]byte, 64)
	rand.Read(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sign(sk, msg)
	}
}

// BenchmarkVerify benchmarks signature verification with a 64-byte message.
func BenchmarkVerify(b *testing.B) {
	seed := make([]byte, 32)
	rand.Read(seed)
	pk, sk := Gen(seed)

	msg := make([]byte, 64)
	rand.Read(msg)
	sig := Sign(sk, msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Verify(pk, msg, sig)
	}
}

// BenchmarkSignVerify benchmarks the full sign+verify cycle.
func BenchmarkSignVerify(b *testing.B) {
	seed := make([]byte, 32)
	rand.Read(seed)
	pk, sk := Gen(seed)

	msg := make([]byte, 64)
	rand.Read(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sig := Sign(sk, msg)
		Verify(pk, msg, sig)
	}
}
