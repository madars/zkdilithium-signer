package field

import (
	"math/bits"
	"testing"
)

var primitiveSink uint32
var primitiveSink64 uint64

const benchInputSize = 1024
const barrettFactor46 uint64 = (uint64(1) << 46) / Q
const barrettMu64 uint64 = (^uint64(0) / Q) + 1
const montRInv uint64 = 12544 // (2^32)^(-1) mod Q

func buildBenchInputs() ([benchInputSize]uint32, [benchInputSize]uint32) {
	var as [benchInputSize]uint32
	var bs [benchInputSize]uint32
	x := uint32(1)
	y := uint32(2)
	for i := 0; i < benchInputSize; i++ {
		x = x*1664525 + 1013904223
		y = y*22695477 + 1
		as[i] = x % Q
		bs[i] = y % Q
	}
	return as, bs
}

func buildBenchInputsLazy() ([benchInputSize]uint32, [benchInputSize]uint32) {
	var as [benchInputSize]uint32
	var bs [benchInputSize]uint32
	x := uint32(1)
	y := uint32(2)
	for i := 0; i < benchInputSize; i++ {
		x = x*1664525 + 1013904223
		y = y*22695477 + 1
		as[i] = x % (2 * Q)
		bs[i] = y % (2 * Q)
	}
	return as, bs
}

// reduceSolinas46 reduces a product p=a*b where a,b < Q (so p < 2^46).
// For Q = 2^23 - 2^20 + 1, use 2^23 == 2^20 - 1 (mod Q) and fold 8 times.
func reduceSolinas46(p uint64) uint32 {
	const mask uint64 = (1 << 23) - 1
	x := p

	h := x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h

	r := uint32(x)
	if r >= Q {
		r -= Q
	}
	return r
}

func reduceMod46(p uint64) uint32 {
	return uint32(p % Q)
}

func reduceBarrett46(p uint64) uint32 {
	hi, lo := bits.Mul64(p, barrettFactor46)
	q := (hi << 18) | (lo >> 46)
	r := p - q*uint64(Q)

	if r >= uint64(Q) {
		r -= uint64(Q)
	}
	if r >= uint64(Q) {
		r -= uint64(Q)
	}
	return uint32(r)
}

func reduceBarrett64(p uint64) uint32 {
	q, _ := bits.Mul64(p, barrettMu64)
	r := p - q*uint64(Q)

	if r >= uint64(Q) {
		r -= uint64(Q)
	}
	if r >= uint64(Q) {
		r -= uint64(Q)
	}
	return uint32(r)
}

func mulSolinas(a, b uint32) uint32 {
	return reduceSolinas46(uint64(a) * uint64(b))
}

func mulSolinasInline(a, b uint32) uint32 {
	const mask uint64 = (1 << 23) - 1
	x := uint64(a) * uint64(b)

	h := x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h
	h = x >> 23
	x = (x & mask) + (h << 20) - h

	r := uint32(x)
	if r >= Q {
		r -= Q
	}
	return r
}

func mulBarrett46(a, b uint32) uint32 {
	return reduceBarrett46(uint64(a) * uint64(b))
}

func mulBarrett64(a, b uint32) uint32 {
	return reduceBarrett64(uint64(a) * uint64(b))
}

// mulMontViaMod computes Montgomery multiplication via %Q backend:
// a*b*R^{-1} mod Q, where R=2^32 and R^{-1}=12544 mod Q.
func mulMontViaMod(a, b uint32) uint32 {
	return uint32((uint64(a) * uint64(b) * montRInv) % Q)
}

func mulMontViaMod2(a, b uint32) uint32 {
	p := (uint64(a) * uint64(b)) % Q
	return uint32((p * montRInv) % Q)
}

// mulMontViaMod1 computes the same Montgomery product using a single modulo:
// (a * b * R^-1) mod Q.
func mulMontViaMod1(a, b uint32) uint32 {
	return mulMontViaMod(a, b)
}

func mulMontViaBarrett(a, b uint32) uint32 {
	p := reduceBarrett64(uint64(a) * uint64(b))
	return reduceBarrett64(uint64(p) * montRInv)
}

func mulMontViaMod4Interleaved(a0, b0, a1, b1, a2, b2, a3, b3 uint32) (r0, r1, r2, r3 uint32) {
	p0 := uint64(a0) * uint64(b0)
	p1 := uint64(a1) * uint64(b1)
	p2 := uint64(a2) * uint64(b2)
	p3 := uint64(a3) * uint64(b3)
	r0 = uint32((p0 * montRInv) % Q)
	r1 = uint32((p1 * montRInv) % Q)
	r2 = uint32((p2 * montRInv) % Q)
	r3 = uint32((p3 * montRInv) % Q)
	return
}

func mulMontViaMod1_4Interleaved(a0, b0, a1, b1, a2, b2, a3, b3 uint32) (r0, r1, r2, r3 uint32) {
	p0 := uint64(a0) * uint64(b0)
	p1 := uint64(a1) * uint64(b1)
	p2 := uint64(a2) * uint64(b2)
	p3 := uint64(a3) * uint64(b3)
	r0 = uint32((p0 * montRInv) % Q)
	r1 = uint32((p1 * montRInv) % Q)
	r2 = uint32((p2 * montRInv) % Q)
	r3 = uint32((p3 * montRInv) % Q)
	return
}

func invSolinas(a uint32) uint32 {
	if a == 0 {
		return 0
	}

	x2 := mulSolinas(a, a)
	x3 := mulSolinas(x2, a)
	x6 := mulSolinas(x3, x3)
	x12 := mulSolinas(x6, x6)
	x15 := mulSolinas(x12, x3)

	res := x6

	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, x15)

	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, x15)

	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, x15)

	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, x15)

	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, res)
	res = mulSolinas(res, x15)

	return res
}

func invBarrett64(a uint32) uint32 {
	if a == 0 {
		return 0
	}

	x2 := mulBarrett64(a, a)
	x3 := mulBarrett64(x2, a)
	x6 := mulBarrett64(x3, x3)
	x12 := mulBarrett64(x6, x6)
	x15 := mulBarrett64(x12, x3)

	res := x6

	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, x15)

	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, x15)

	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, x15)

	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, x15)

	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, res)
	res = mulBarrett64(res, x15)

	return res
}

func TestReductionCandidates(t *testing.T) {
	x := uint32(1)
	y := uint32(2)

	for i := 0; i < 200000; i++ {
		x = x*1664525 + 1013904223
		y = y*22695477 + 1
		a := x % Q
		b := y % Q
		p := uint64(a) * uint64(b)
		want := uint32(p % Q)

		if got := reduceSolinas46(p); got != want {
			t.Fatalf("reduceSolinas46 mismatch: a=%d b=%d got=%d want=%d", a, b, got, want)
		}
		if got := reduceMod46(p); got != want {
			t.Fatalf("reduceMod46 mismatch: a=%d b=%d got=%d want=%d", a, b, got, want)
		}
		if got := reduceBarrett46(p); got != want {
			t.Fatalf("reduceBarrett46 mismatch: a=%d b=%d got=%d want=%d", a, b, got, want)
		}
		if got := reduceBarrett64(p); got != want {
			t.Fatalf("reduceBarrett64 mismatch: a=%d b=%d got=%d want=%d", a, b, got, want)
		}
		if got := mulMontViaMod(a, b); got != mulMontViaBarrett(a, b) {
			t.Fatalf("mulMontViaMod vs mulMontViaBarrett mismatch: a=%d b=%d got=%d want=%d", a, b, got, mulMontViaBarrett(a, b))
		}
	}
}

func BenchmarkPrimitiveAdd(b *testing.B) {
	x := uint32(1)
	y := uint32(2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x = x*1664525 + 1013904223
		y = y*22695477 + 1
		primitiveSink = Add(x%Q, y%Q)
	}
}

func BenchmarkPrimitiveSub(b *testing.B) {
	x := uint32(1)
	y := uint32(2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x = x*1664525 + 1013904223
		y = y*22695477 + 1
		primitiveSink = Sub(x%Q, y%Q)
	}
}

func BenchmarkPrimitiveReduce(b *testing.B) {
	x := uint32(1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x = x*1664525 + 1013904223
		primitiveSink = reduce(x % (2 * Q))
	}
}

func BenchmarkPrimitiveInvPlainLazyInput(b *testing.B) {
	as, _ := buildBenchInputsLazy()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = invPlainLazy(as[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveMulMod(b *testing.B) {
	as, bs := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = Mul(as[idx], bs[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveMulSolinas(b *testing.B) {
	as, bs := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = mulSolinas(as[idx], bs[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveMulSolinasInline(b *testing.B) {
	as, bs := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = mulSolinasInline(as[idx], bs[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveMulBarrett46(b *testing.B) {
	as, bs := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = mulBarrett46(as[idx], bs[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveMulBarrett64(b *testing.B) {
	as, bs := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = mulBarrett64(as[idx], bs[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveMulMontViaMod(b *testing.B) {
	as, bs := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = mulMontViaMod(as[idx], bs[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveMulMontViaMod2(b *testing.B) {
	as, bs := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = mulMontViaMod2(as[idx], bs[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveMulMontViaBarrett(b *testing.B) {
	as, bs := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = mulMontViaBarrett(as[idx], bs[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveMulMontViaMod1(b *testing.B) {
	as, bs := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = mulMontViaMod1(as[idx], bs[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveMulPlainLazy2Inputs(b *testing.B) {
	as, bs := buildBenchInputs()
	idx := 0
	var s0, s1 uint32
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := idx + 1
		if j >= benchInputSize {
			j = 0
		}
		s0, s1 = mulPlainLazy2(as[idx], bs[idx], as[j], bs[j])
		idx += 2
		if idx >= benchInputSize {
			idx -= benchInputSize
		}
	}
	primitiveSink = s0 ^ s1
}

func BenchmarkKernelChain1MulMontViaMod(b *testing.B) {
	as, bs := buildBenchInputs()
	x := as[0]
	idx := 1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x = mulMontViaMod(x, bs[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
	primitiveSink = x
}

func BenchmarkKernelChain4MulMontViaMod(b *testing.B) {
	as, bs := buildBenchInputs()
	x0, x1, x2, x3 := as[0], as[1], as[2], as[3]
	idx := 4
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x0 = mulMontViaMod(x0, bs[idx+0])
		x1 = mulMontViaMod(x1, bs[idx+1])
		x2 = mulMontViaMod(x2, bs[idx+2])
		x3 = mulMontViaMod(x3, bs[idx+3])
		idx += 4
		if idx+3 >= benchInputSize {
			idx = 0
		}
	}
	primitiveSink = x0 + x1 + x2 + x3
}

func BenchmarkKernelChain4MulMontViaMod4Interleaved(b *testing.B) {
	as, bs := buildBenchInputs()
	x0, x1, x2, x3 := as[0], as[1], as[2], as[3]
	idx := 4
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x0, x1, x2, x3 = mulMontViaMod4Interleaved(
			x0, bs[idx+0],
			x1, bs[idx+1],
			x2, bs[idx+2],
			x3, bs[idx+3],
		)
		idx += 4
		if idx+3 >= benchInputSize {
			idx = 0
		}
	}
	primitiveSink = x0 + x1 + x2 + x3
}

func BenchmarkKernelChain1MulMontViaMod1(b *testing.B) {
	as, bs := buildBenchInputs()
	x := as[0]
	idx := 1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x = mulMontViaMod1(x, bs[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
	primitiveSink = x
}

func BenchmarkKernelChain4MulMontViaMod1(b *testing.B) {
	as, bs := buildBenchInputs()
	x0, x1, x2, x3 := as[0], as[1], as[2], as[3]
	idx := 4
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x0 = mulMontViaMod1(x0, bs[idx+0])
		x1 = mulMontViaMod1(x1, bs[idx+1])
		x2 = mulMontViaMod1(x2, bs[idx+2])
		x3 = mulMontViaMod1(x3, bs[idx+3])
		idx += 4
		if idx+3 >= benchInputSize {
			idx = 0
		}
	}
	primitiveSink = x0 + x1 + x2 + x3
}

func BenchmarkKernelChain4MulMontViaMod1Interleaved(b *testing.B) {
	as, bs := buildBenchInputs()
	x0, x1, x2, x3 := as[0], as[1], as[2], as[3]
	idx := 4
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x0, x1, x2, x3 = mulMontViaMod1_4Interleaved(
			x0, bs[idx+0],
			x1, bs[idx+1],
			x2, bs[idx+2],
			x3, bs[idx+3],
		)
		idx += 4
		if idx+3 >= benchInputSize {
			idx = 0
		}
	}
	primitiveSink = x0 + x1 + x2 + x3
}

func BenchmarkKernelPrefix35MulMontViaMod1(b *testing.B) {
	as, _ := buildBenchInputs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := as[0]
		for j := 1; j < 35; j++ {
			p = mulMontViaMod1(p, as[j])
		}
		primitiveSink = p
	}
}

func BenchmarkKernelPrefix35MulMontViaMod(b *testing.B) {
	as, _ := buildBenchInputs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := as[0]
		for j := 1; j < 35; j++ {
			p = mulMontViaMod(p, as[j])
		}
		primitiveSink = p
	}
}

func BenchmarkPrimitiveReduceSolinas46(b *testing.B) {
	as, bs := buildBenchInputs()
	var prods [benchInputSize]uint64
	for i := 0; i < benchInputSize; i++ {
		prods[i] = uint64(as[i]) * uint64(bs[i])
	}
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = reduceSolinas46(prods[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveReduceMod46(b *testing.B) {
	as, bs := buildBenchInputs()
	var prods [benchInputSize]uint64
	for i := 0; i < benchInputSize; i++ {
		prods[i] = uint64(as[i]) * uint64(bs[i])
	}
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = reduceMod46(prods[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveReduceBarrett46(b *testing.B) {
	as, bs := buildBenchInputs()
	var prods [benchInputSize]uint64
	for i := 0; i < benchInputSize; i++ {
		prods[i] = uint64(as[i]) * uint64(bs[i])
	}
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = reduceBarrett46(prods[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveReduceBarrett64(b *testing.B) {
	as, bs := buildBenchInputs()
	var prods [benchInputSize]uint64
	for i := 0; i < benchInputSize; i++ {
		prods[i] = uint64(as[i]) * uint64(bs[i])
	}
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		primitiveSink = reduceBarrett64(prods[idx])
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveInvSolinas(b *testing.B) {
	as, _ := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := as[idx]
		if v == 0 {
			v = 1
		}
		primitiveSink = invSolinas(v)
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}

func BenchmarkPrimitiveInvBarrett64(b *testing.B) {
	as, _ := buildBenchInputs()
	idx := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := as[idx]
		if v == 0 {
			v = 1
		}
		primitiveSink = invBarrett64(v)
		idx++
		if idx == benchInputSize {
			idx = 0
		}
	}
}
