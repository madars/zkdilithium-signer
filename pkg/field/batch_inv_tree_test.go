package field

import (
	"testing"
)

func TestBatchInvMontTree(t *testing.T) {
	// Test with various sizes
	sizes := []int{1, 2, 3, 4, 5, 7, 8, 15, 16, 31, 32, 35, 63, 64}

	for _, n := range sizes {
		t.Run("size_"+string(rune('0'+n/10))+string(rune('0'+n%10)), func(t *testing.T) {
			// Generate test inputs
			xs := make([]uint32, n)
			for i := 0; i < n; i++ {
				xs[i] = ToMont(uint32(i + 1)) // 1, 2, 3, ...
			}

			// Make copies for both methods
			xs1 := make([]uint32, n)
			xs2 := make([]uint32, n)
			copy(xs1, xs)
			copy(xs2, xs)

			// Scratch space
			scratch1 := make([]uint32, n)
			scratch2 := make([]uint32, 3*n) // Tree needs ~2n for layers, extra for safety

			// Run both methods
			BatchInvMontLinear(xs1, scratch1)
			BatchInvMontTreeNoZero(xs2, scratch2)

			// Compare results
			for i := 0; i < n; i++ {
				if xs1[i] != xs2[i] {
					t.Errorf("index %d: linear=%d, tree=%d", i, xs1[i], xs2[i])
				}
			}

			// Verify correctness: x * x^-1 should equal 1
			for i := 0; i < n; i++ {
				orig := ToMont(uint32(i + 1))
				inv := xs2[i]
				prod := MulMont(orig, inv)
				if prod != ToMont(1) {
					t.Errorf("index %d: %d * %d = %d, expected 1_M=%d",
						i, orig, inv, prod, ToMont(1))
				}
			}
		})
	}
}

func TestBatchInvMontTreeNoZeroILP(t *testing.T) {
	// Test ILP version matches non-ILP version
	sizes := []int{1, 2, 3, 4, 5, 7, 8, 15, 16, 31, 32, 35, 63, 64}

	for _, n := range sizes {
		xs1 := make([]uint32, n)
		xs2 := make([]uint32, n)
		for i := 0; i < n; i++ {
			xs1[i] = ToMont(uint32(i + 1))
			xs2[i] = ToMont(uint32(i + 1))
		}

		scratch1 := make([]uint32, 3*n)
		scratch2 := make([]uint32, 3*n)

		BatchInvMontTreeNoZero(xs1, scratch1)
		BatchInvMontTreeNoZeroILP(xs2, scratch2)

		for i := 0; i < n; i++ {
			if xs1[i] != xs2[i] {
				t.Errorf("size %d, index %d: NoZero=%d, ILP=%d", n, i, xs1[i], xs2[i])
			}
		}
	}
}

func TestBatchInvMontTreeNoZeroILP4(t *testing.T) {
	// Test ILP4 version matches non-ILP version
	sizes := []int{1, 2, 3, 4, 5, 7, 8, 15, 16, 31, 32, 35, 63, 64}

	for _, n := range sizes {
		xs1 := make([]uint32, n)
		xs2 := make([]uint32, n)
		for i := 0; i < n; i++ {
			xs1[i] = ToMont(uint32(i + 1))
			xs2[i] = ToMont(uint32(i + 1))
		}

		scratch1 := make([]uint32, 3*n)
		scratch2 := make([]uint32, 3*n)

		BatchInvMontTreeNoZero(xs1, scratch1)
		BatchInvMontTreeNoZeroILP4(xs2, scratch2)

		for i := 0; i < n; i++ {
			if xs1[i] != xs2[i] {
				t.Errorf("size %d, index %d: NoZero=%d, ILP4=%d", n, i, xs1[i], xs2[i])
			}
		}
	}
}

func TestBatchInvMontTreeWithZeros(t *testing.T) {
	n := 35
	xs := make([]uint32, n)
	for i := 0; i < n; i++ {
		xs[i] = ToMont(uint32(i + 1))
	}
	// Insert some zeros
	xs[0] = 0
	xs[17] = 0
	xs[34] = 0

	// Make copies
	xs1 := make([]uint32, n)
	xs2 := make([]uint32, n)
	copy(xs1, xs)
	copy(xs2, xs)

	scratch1 := make([]uint32, n)
	scratch2 := make([]uint32, 3*n)

	BatchInvMontLinear(xs1, scratch1)
	BatchInvMontTree(xs2, scratch2)

	for i := 0; i < n; i++ {
		if xs1[i] != xs2[i] {
			t.Errorf("index %d: linear=%d, tree=%d", i, xs1[i], xs2[i])
		}
	}
}

func BenchmarkBatchInvLinear35(b *testing.B) {
	xs := make([]uint32, 35)
	scratch := make([]uint32, 35)
	for i := 0; i < 35; i++ {
		xs[i] = ToMont(uint32(i + 1))
	}
	orig := make([]uint32, 35)
	copy(orig, xs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(xs, orig)
		BatchInvMontLinear(xs, scratch)
	}
}

func BenchmarkBatchInvTree35(b *testing.B) {
	xs := make([]uint32, 35)
	scratch := make([]uint32, 128) // Extra space for tree
	for i := 0; i < 35; i++ {
		xs[i] = ToMont(uint32(i + 1))
	}
	orig := make([]uint32, 35)
	copy(orig, xs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(xs, orig)
		BatchInvMontTree(xs, scratch)
	}
}

func BenchmarkBatchInvTreeNoZero35(b *testing.B) {
	xs := make([]uint32, 35)
	scratch := make([]uint32, 128)
	for i := 0; i < 35; i++ {
		xs[i] = ToMont(uint32(i + 1))
	}
	orig := make([]uint32, 35)
	copy(orig, xs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(xs, orig)
		BatchInvMontTreeNoZero(xs, scratch)
	}
}

func BenchmarkBatchInvTreeCond35(b *testing.B) {
	xs := make([]uint32, 35)
	scratch := make([]uint32, 128)
	for i := 0; i < 35; i++ {
		xs[i] = ToMont(uint32(i + 1))
	}
	orig := make([]uint32, 35)
	copy(orig, xs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(xs, orig)
		BatchInvMontTreeCond(xs, scratch)
	}
}

func BenchmarkBatchInvTreeNoZeroILP35(b *testing.B) {
	xs := make([]uint32, 35)
	scratch := make([]uint32, 128)
	for i := 0; i < 35; i++ {
		xs[i] = ToMont(uint32(i + 1))
	}
	orig := make([]uint32, 35)
	copy(orig, xs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(xs, orig)
		BatchInvMontTreeNoZeroILP(xs, scratch)
	}
}

func BenchmarkBatchInvTreeNoZeroILP4_35(b *testing.B) {
	xs := make([]uint32, 35)
	scratch := make([]uint32, 128)
	for i := 0; i < 35; i++ {
		xs[i] = ToMont(uint32(i + 1))
	}
	orig := make([]uint32, 35)
	copy(orig, xs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(xs, orig)
		BatchInvMontTreeNoZeroILP4(xs, scratch)
	}
}

// Benchmark with larger sizes
func BenchmarkBatchInvLinear256(b *testing.B) {
	xs := make([]uint32, 256)
	scratch := make([]uint32, 256)
	for i := 0; i < 256; i++ {
		xs[i] = ToMont(uint32(i + 1))
	}
	orig := make([]uint32, 256)
	copy(orig, xs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(xs, orig)
		BatchInvMontLinear(xs, scratch)
	}
}

func BenchmarkBatchInvTreeNoZero256(b *testing.B) {
	xs := make([]uint32, 256)
	scratch := make([]uint32, 512)
	for i := 0; i < 256; i++ {
		xs[i] = ToMont(uint32(i + 1))
	}
	orig := make([]uint32, 256)
	copy(orig, xs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(xs, orig)
		BatchInvMontTreeNoZero(xs, scratch)
	}
}
