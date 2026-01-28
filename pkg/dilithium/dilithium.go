// Package dilithium implements zkDilithium signature scheme.
package dilithium

import (
	"zkdilithium-signer/pkg/encoding"
	"zkdilithium-signer/pkg/field"
	"zkdilithium-signer/pkg/hash"
	"zkdilithium-signer/pkg/ntt"
	"zkdilithium-signer/pkg/poly"
	"zkdilithium-signer/pkg/sampling"
)

// Gen generates a keypair from a 32-byte seed.
func Gen(seed []byte) (pk, sk []byte) {
	if len(seed) != 32 {
		panic("seed must be 32 bytes")
	}

	// Expand seed
	expandedSeed := hash.H(seed, 32+64+32)
	rho := expandedSeed[:32]
	rho2 := expandedSeed[32 : 32+64]
	key := expandedSeed[32+64:]

	// Sample matrix A in NTT domain (normal form)
	Ahat := sampling.SampleMatrix(rho)

	// Sample secret vectors (normal form)
	s1, s2 := sampling.SampleSecret(rho2)

	// Precompute NTT of s1 (normal form)
	var s1Hat [field.L]poly.Poly
	for j := 0; j < field.L; j++ {
		s1Hat[j] = s1[j]
		s1Hat[j].NTT()
	}

	// Compute t = A*s1 + s2 (all in normal form, no Montgomery)
	var t [field.K]poly.Poly
	for i := 0; i < field.K; i++ {
		var sum poly.Poly
		for j := 0; j < field.L; j++ {
			var prod poly.Poly
			ntt.MulNTT((*[field.N]uint32)(&Ahat[i][j]), (*[field.N]uint32)(&s1Hat[j]), (*[field.N]uint32)(&prod))
			poly.Add(&sum, &prod, &sum)
		}
		sum.InvNTT()
		poly.Add(&sum, &s2[i], &t[i])
	}

	// Pack t
	tPacked := make([]byte, 0, field.K*field.N*3)
	for i := 0; i < field.K; i++ {
		tPacked = append(tPacked, encoding.PackPoly((*[field.N]uint32)(&t[i]))...)
	}

	// Compute tr = H(rho || tPacked)
	// Note: we build a fresh buffer to avoid aliasing issues if rho has spare capacity
	trInput := make([]byte, len(rho)+len(tPacked))
	copy(trInput, rho)
	copy(trInput[len(rho):], tPacked)
	tr := hash.H(trInput, 32)

	// s1, s2 are already in normal form (no Montgomery conversion)

	// Pack s1 and s2
	s1Packed := make([]byte, 0, field.L*96)
	for i := 0; i < field.L; i++ {
		s1Packed = append(s1Packed, encoding.PackPolyLeqEta((*[field.N]uint32)(&s1[i]))...)
	}
	s2Packed := make([]byte, 0, field.K*96)
	for i := 0; i < field.K; i++ {
		s2Packed = append(s2Packed, encoding.PackPolyLeqEta((*[field.N]uint32)(&s2[i]))...)
	}

	// Public key: rho || tPacked
	pk = make([]byte, len(rho)+len(tPacked))
	copy(pk, rho)
	copy(pk[len(rho):], tPacked)

	// Secret key: rho || key || tr || s1Packed || s2Packed || tPacked
	sk = make([]byte, 0, 32+32+32+96*field.L+96*field.K+field.K*field.N*3)
	sk = append(sk, rho...)
	sk = append(sk, key...)
	sk = append(sk, tr...)
	sk = append(sk, s1Packed...)
	sk = append(sk, s2Packed...)
	sk = append(sk, tPacked...)

	return pk, sk
}

// Sign signs a message with the secret key.
func Sign(sk, msg []byte) []byte {
	// Unpack secret key
	rho := sk[:32]
	key := sk[32:64]
	tr := sk[64:96]

	// Unpack s1, convert to Montgomery form
	var s1 [field.L]poly.Poly
	for i := 0; i < field.L; i++ {
		s1[i] = encoding.UnpackPolyLeqEta(sk[96+i*96 : 96+(i+1)*96])
		s1[i].ToMont()
	}

	// Unpack s2, convert to Montgomery form
	var s2 [field.K]poly.Poly
	for i := 0; i < field.K; i++ {
		s2[i] = encoding.UnpackPolyLeqEta(sk[96+96*field.L+i*96 : 96+96*field.L+(i+1)*96])
		s2[i].ToMont()
	}

	// Sample matrix A, convert to Montgomery form
	Ahat := sampling.SampleMatrix(rho)
	for i := 0; i < field.K; i++ {
		for j := 0; j < field.L; j++ {
			Ahat[i][j].ToMont()
		}
	}

	// Compute mu using Poseidon
	hMu := hash.NewPoseidon([]uint32{0})
	hMu.Write(encoding.BytesToFes(tr))
	hMu.Permute()
	hMu.Write(encoding.BytesToFes(msg))
	mu := hMu.Read(field.MuSize)

	// Precompute NTT of secrets (Montgomery form)
	var s1Hat [field.L]poly.Poly
	for i := 0; i < field.L; i++ {
		s1Hat[i] = s1[i]
		s1Hat[i].NTT()
	}
	var s2Hat [field.K]poly.Poly
	for i := 0; i < field.K; i++ {
		s2Hat[i] = s2[i]
		s2Hat[i].NTT()
	}

	// Derive rho2 for y sampling
	trMsg := make([]byte, len(tr)+len(msg))
	copy(trMsg, tr)
	copy(trMsg[len(tr):], msg)
	innerHash := hash.H(trMsg, 64)
	keyHash := make([]byte, len(key)+len(innerHash))
	copy(keyHash, key)
	copy(keyHash[len(key):], innerHash)
	rho2 := hash.H(keyHash, 64)

	yNonce := 0
	for {
		// Sample y, convert to Montgomery form and NTT
		y := sampling.SampleY(rho2, yNonce)
		yNonce += field.L
		var yHat [field.L]poly.Poly
		for i := 0; i < field.L; i++ {
			y[i].ToMont()
			yHat[i] = y[i]
			yHat[i].NTT()
		}

		// Compute w = A * y using lazy accumulation (Montgomery form)
		var wMont [field.K]poly.Poly
		poly.MatVecMulNTTLazy(&Ahat, &yHat, &wMont)
		for i := 0; i < field.K; i++ {
			wMont[i].InvNTT()
		}

		// Convert w from Montgomery for Decompose
		var w [field.K]poly.Poly
		for i := 0; i < field.K; i++ {
			w[i] = wMont[i]
			w[i].FromMont()
		}

		// Decompose w
		var w1 [field.K]poly.Poly
		for i := 0; i < field.K; i++ {
			_, w1[i] = w[i].Decompose()
		}

		// Compute challenge hash
		hC := hash.NewPoseidon(nil)
		hC.Write(mu)
		for j := 0; j < field.N; j++ {
			for i := 0; i < field.K; i++ {
				hC.Write([]uint32{w1[i][j]})
			}
		}
		cTilde := hC.Read(field.CSize)

		// Sample c from cTilde, convert to Montgomery form
		hBall := hash.NewPoseidon(append([]uint32{2}, cTilde...))
		c := sampling.SampleInBall(hBall)
		if c == nil {
			continue // Rejection
		}
		c.ToMont()

		// Compute cs2 = c * s2 (in NTT domain, Montgomery form)
		var cHat poly.Poly = *c
		cHat.NTT()

		var cs2Mont [field.K]poly.Poly
		for i := 0; i < field.K; i++ {
			poly.MulNTT(&cHat, &s2Hat[i], &cs2Mont[i])
			cs2Mont[i].InvNTT()
		}

		// r0 = w - cs2 (both need to be in same form)
		// wMont and cs2Mont are both in Montgomery form
		var r0Mont [field.K]poly.Poly
		for i := 0; i < field.K; i++ {
			poly.Sub(&wMont[i], &cs2Mont[i], &r0Mont[i])
		}

		// Convert r0 from Montgomery for Decompose and Norm
		var r0 [field.K]poly.Poly
		for i := 0; i < field.K; i++ {
			r0[i] = r0Mont[i]
			r0[i].FromMont()
		}

		r0Decomposed := make([][field.N]uint32, field.K)
		for i := 0; i < field.K; i++ {
			r0Decomposed[i], _ = r0[i].Decompose()
		}

		// Check norm of r0
		var maxR0Norm uint32
		for i := 0; i < field.K; i++ {
			var p poly.Poly = r0Decomposed[i]
			n := p.Norm()
			if n > maxR0Norm {
				maxR0Norm = n
			}
		}
		if maxR0Norm >= field.Gamma2-field.Beta {
			continue
		}

		// Compute z = y + c*s1 (Montgomery form)
		var zMont [field.L]poly.Poly
		for i := 0; i < field.L; i++ {
			var cs1 poly.Poly
			poly.MulNTT(&cHat, &s1Hat[i], &cs1)
			cs1.InvNTT()
			poly.Add(&y[i], &cs1, &zMont[i])
		}

		// Convert z from Montgomery for Norm check and packing
		var z [field.L]poly.Poly
		for i := 0; i < field.L; i++ {
			z[i] = zMont[i]
			z[i].FromMont()
		}

		// Check norm of z
		var maxZNorm uint32
		for i := 0; i < field.L; i++ {
			n := z[i].Norm()
			if n > maxZNorm {
				maxZNorm = n
			}
		}
		if maxZNorm >= field.Gamma1-field.Beta {
			continue
		}

		// Pack signature (z is already in normal form)
		sig := encoding.PackFes(cTilde)
		for i := 0; i < field.L; i++ {
			sig = append(sig, encoding.PackPolyLeGamma1((*[field.N]uint32)(&z[i]))...)
		}
		return sig
	}
}

// Verify verifies a signature.
func Verify(pk, msg, sig []byte) bool {
	expectedSigLen := field.CSize*3 + field.PolyLeGamma1Size*field.L
	if len(sig) != expectedSigLen {
		return false
	}

	// Unpack signature
	packedCTilde := sig[:field.CSize*3]
	packedZ := sig[field.CSize*3:]
	cTilde := encoding.UnpackFes(packedCTilde)

	// Unpack z (normal form for norm check)
	var z [field.L]poly.Poly
	for i := 0; i < field.L; i++ {
		z[i] = encoding.UnpackPolyLeGamma1(packedZ[i*field.PolyLeGamma1Size : (i+1)*field.PolyLeGamma1Size])
	}

	// Check z norm (before converting to Montgomery)
	for i := 0; i < field.L; i++ {
		if z[i].Norm() >= field.Gamma1-field.Beta {
			return false
		}
	}

	// Convert z to Montgomery form for NTT operations
	var zMont [field.L]poly.Poly
	for i := 0; i < field.L; i++ {
		zMont[i] = z[i]
		zMont[i].ToMont()
	}

	// Unpack public key
	rho := pk[:32]
	tPacked := pk[32:]

	// Unpack t, convert to Montgomery form
	var tMont [field.K]poly.Poly
	for i := 0; i < field.K; i++ {
		tMont[i] = encoding.UnpackPoly(tPacked[i*field.N*3 : (i+1)*field.N*3])
		tMont[i].ToMont()
	}

	// Compute tr
	tr := hash.H(pk, 32)

	// Compute mu
	hMu := hash.NewPoseidon([]uint32{0})
	hMu.Write(encoding.BytesToFes(tr))
	hMu.Permute()
	hMu.Write(encoding.BytesToFes(msg))
	mu := hMu.Read(field.MuSize)

	// Sample c from cTilde, convert to Montgomery form
	hBall := hash.NewPoseidon(append([]uint32{2}, cTilde...))
	c := sampling.SampleInBall(hBall)
	if c == nil {
		return false
	}
	c.ToMont()

	// Sample A, convert to Montgomery form
	Ahat := sampling.SampleMatrix(rho)
	for i := 0; i < field.K; i++ {
		for j := 0; j < field.L; j++ {
			Ahat[i][j].ToMont()
		}
	}

	// Compute Az - tc in NTT domain (Montgomery form)
	var cHat poly.Poly = *c
	cHat.NTT()

	var zHat [field.L]poly.Poly
	for i := 0; i < field.L; i++ {
		zHat[i] = zMont[i]
		zHat[i].NTT()
	}

	var tHat [field.K]poly.Poly
	for i := 0; i < field.K; i++ {
		tHat[i] = tMont[i]
		tHat[i].NTT()
	}

	// Compute Az using lazy accumulation
	var Az [field.K]poly.Poly
	poly.MatVecMulNTTLazy(&Ahat, &zHat, &Az)

	// Compute w1 = Az - tc for each row
	var w1 [field.K]poly.Poly
	for i := 0; i < field.K; i++ {
		// tc (Montgomery form)
		var tc poly.Poly
		poly.MulNTT(&tHat[i], &cHat, &tc)

		// Az - tc (Montgomery form)
		poly.Sub(&Az[i], &tc, &Az[i])
		Az[i].InvNTT()

		// Convert from Montgomery for Decompose
		Az[i].FromMont()

		// Decompose
		_, w1[i] = Az[i].Decompose()
	}

	// Recompute challenge
	hC := hash.NewPoseidon(nil)
	hC.Write(mu)
	for j := 0; j < field.N; j++ {
		for i := 0; i < field.K; i++ {
			hC.Write([]uint32{w1[i][j]})
		}
	}
	cTilde2 := hC.Read(field.CSize)

	// Compare
	for i := range cTilde {
		if cTilde[i] != cTilde2[i] {
			return false
		}
	}
	return true
}
