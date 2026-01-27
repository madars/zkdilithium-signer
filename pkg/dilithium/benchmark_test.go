package dilithium

import (
	"fmt"
	"testing"
)

// Fixed seed for deterministic benchmarks.
// Using a fixed seed ensures reproducible timing across runs.
var benchSeed = []byte{
	0x80, 0xfb, 0xd4, 0xab, 0x13, 0x16, 0xa3, 0x25,
	0x33, 0x95, 0x65, 0x67, 0x38, 0x6e, 0xdf, 0x85,
	0x1a, 0x15, 0x4a, 0x71, 0x4a, 0x4d, 0x2a, 0xa7,
	0x7d, 0xbc, 0x85, 0xb1, 0x76, 0xcb, 0x88, 0xd4,
}

// benchMsgs contains varied messages to capture rejection sampling variance.
// Pre-generated to avoid allocation during benchmark.
var benchMsgs [][]byte

func init() {
	benchMsgs = make([][]byte, 1000)
	for i := range benchMsgs {
		benchMsgs[i] = []byte(fmt.Sprintf("benchmark message %d", i))
	}
}

// BenchmarkGen benchmarks key generation.
func BenchmarkGen(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Gen(benchSeed)
	}
}

// BenchmarkSign benchmarks signing with varied messages to capture
// rejection sampling variance (different messages lead to different
// numbers of rejection sampling iterations).
func BenchmarkSign(b *testing.B) {
	_, sk := Gen(benchSeed)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sign(sk, benchMsgs[i%len(benchMsgs)])
	}
}

// BenchmarkVerify benchmarks signature verification.
func BenchmarkVerify(b *testing.B) {
	pk, sk := Gen(benchSeed)
	sig := Sign(sk, benchMsgs[0])

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Verify(pk, benchMsgs[0], sig)
	}
}

// BenchmarkSignVerify benchmarks the full sign+verify cycle with varied messages.
func BenchmarkSignVerify(b *testing.B) {
	pk, sk := Gen(benchSeed)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := benchMsgs[i%len(benchMsgs)]
		sig := Sign(sk, msg)
		Verify(pk, msg, sig)
	}
}
