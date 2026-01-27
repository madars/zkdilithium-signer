package dilithium

import (
	"encoding/hex"
	"testing"
)

// Test key generation sizes match Python
func TestGenKeySizes(t *testing.T) {
	pk, sk := Gen(make([]byte, 32))
	if len(pk) != 3104 {
		t.Errorf("pk length = %d, want 3104", len(pk))
	}
	if len(sk) != 3936 {
		t.Errorf("sk length = %d, want 3936", len(sk))
	}
}

// Test key generation pk prefix matches Python
func TestGenPKPrefix(t *testing.T) {
	pk, _ := Gen(make([]byte, 32))
	expected, _ := hex.DecodeString("f5977c8283546a63723bc31d2619124f11db4658643336741df81757d5ad3062")
	for i := 0; i < 32; i++ {
		if pk[i] != expected[i] {
			t.Errorf("pk[%d] = %02x, want %02x", i, pk[i], expected[i])
		}
	}
}

// Test key generation pk bytes 32-64 match Python
func TestGenPK32to64(t *testing.T) {
	pk, _ := Gen(make([]byte, 32))
	expected, _ := hex.DecodeString("70bb4539916e957a230b273b97c81553730160dd5f59520815b15c2500087705")
	for i := 0; i < 32; i++ {
		if pk[32+i] != expected[i] {
			t.Errorf("pk[%d] = %02x, want %02x", 32+i, pk[32+i], expected[i])
		}
	}
}

// Test key generation is deterministic
func TestGenDeterministic(t *testing.T) {
	seed := make([]byte, 32)
	seed[0] = 0x09
	pk1, sk1 := Gen(seed)
	pk2, sk2 := Gen(seed)

	if string(pk1) != string(pk2) {
		t.Error("Gen not deterministic: pk differs")
	}
	if string(sk1) != string(sk2) {
		t.Error("Gen not deterministic: sk differs")
	}
}

// Test signature size
func TestSignSize(t *testing.T) {
	_, sk := Gen(make([]byte, 32))
	sig := Sign(sk, []byte("test"))
	if len(sig) != 2340 {
		t.Errorf("sig length = %d, want 2340", len(sig))
	}
}

// THE GOLD STANDARD TEST
// Test full signature matches Python byte-for-byte
func TestSignFullSignature(t *testing.T) {
	_, sk := Gen(make([]byte, 32))
	sig := Sign(sk, []byte("test"))

	expectedHex := "58044167cc06656d0434e13745b66aacdc0642b03eda3361239105618a292dc8" +
		"6cbb5f06b3e1abc9ec5243404288a88f212756e3cfd58b6bbc1660a935a11346" +
		"315559f1186a10d538c6c9a062dadd9f27457c7a94b7f33e2651a1fbee0730eb" +
		"7dba2aa94fc8fced86bdb9301c67175893708124d1f5744aa047b9711f29335e" +
		"46e0734497d3be5863fb9b04ffb06789df7ff7d6cd7199d4c0da514e9ab8d3e0" +
		"fafc306ca63484f890b0d5c2ddc9a79fbd5780e404b5404dc6ad3b67603781a6" +
		"5ec579d3c3d88d05f8367fec9af15157b68c070bf6217e18947f016a29aa17d1" +
		"337c484edec8cd00b502e80c910c27bd818840116417d63230bf06ac28c14e71" +
		"90347db275326c857431e01694f7c2fdec2c0d5044521aaa23ecec21d9062b3a" +
		"be58fe85f52e481ef2c851bc4590374d87c04ca398a2b873683a326ef110f4c3" +
		"2d1d9f0f61e9ae4f916752253a8c636dbf92e312e74d7b35f9e4723b78cfaae9" +
		"acb435e8f653ab96b1e6044247b6ddb2bbd9fe5993eb07b24edb8849a80d853b" +
		"5e44e2cc573cd53f74b89b72930770f934ec2e55c7a9a9e90a5c639f99bcbb37" +
		"ded70e157054d18b55cf62da928c027ca40543dcba2f8402020f6e853bad08ce" +
		"ce700c56df0efe32226383fd827c31033be01adfa19b41d2bf855bc081d450db" +
		"d46a4ddc12cbcd4a7e27ff48076f19342f9b1344625fbbe557c1215748abdbf6" +
		"4b00d6cc9c17519a725c90accc3abb4939a800a847dc92bb77764e5795eea103" +
		"bfe284b5dae13028b258c4e42310275485d83372e7ec5f931a8b9eb5387afc51" +
		"90dc73665991a7aedc11281c9bcf986ad90d5a96db50ab762ef37adb9f635478" +
		"5c4e0fc823a05378de71f6112aff61475fca5e17c034ffb4fe7b3330015fc7b2" +
		"f80909e8e0365dc93635869bb6ceb9940c1b36796a097cce205de0b5a43d8c1d" +
		"76263eabc72bdb2bc2d19e03ad3a2cabc702965b6e186629cf94963455456730" +
		"088bbf656388825e7bc1ef14d9ad428bb8d204a9e7d665163a06518cdd16d688" +
		"5fbea215c6e30e60a98766931cfcacad9793ea64e6b71a4b6bb4fb7d24cfcfa8" +
		"de0ece9e794173195592caa615659221407a8daa5dffac94861dea2c960a608b" +
		"58b490fcff361661727bd45c9d0bf6907cc07176cc6304d27119f3c8aaa24351" +
		"87e88f35082ff43dd9e8ed82b7f0c6882df00ac0c59f2c9f739ea3e36513830f" +
		"b758f244ad5fffe55a18f633b76a2b697ab1b09c90b8f07f35355b23ebecb55a" +
		"d987f6b31334525af1bfa0fe4d6756fdbf24de831725201f87d93ba6b370aaf2" +
		"028050b09b855d1f5c3e8ed1f3e82fe0315c28bf26a0078cb2d2570d4c18f591" +
		"ebc78744815d334601b26efb56ec9d8e4c719ef8eb3cf2aa97168449877610e3" +
		"fb4a4d683b54741e2c24cc67d6f8822ba9e26bfff8cc626de1a41b2309b3ee1f" +
		"88fd2cf9bc122b3c085bfef553310991d6e1f75d04f1e1192451c383b4b6e525" +
		"9a1bfba1f4024915e5dc97dffdf9a8de4a1925b1c71adee0415e5989cc3cf1ed" +
		"185a99c69835cbd7e44e873e9b4ef0b9ac20e99e733fb78387b31d020c41ded2" +
		"90e61cbe3d34881a44b21b308a46b964799338cba84514ccb7d913cb0be3c658" +
		"95abb22764a78ac1d8b97de23dea60ca075bf6cc2d7e63a515589756579242e7" +
		"fec1f954ec04da9db5447f4b6af584e6b1f9068d1388baea3785895b5bc6a4fc" +
		"f8ee9daaa8b58e0bfc541182be3b5f86d916bf245bacffe59d0e45ded99554ec" +
		"5ba11e0556120290d848da3ef6e1f6a4e320c76b7dd94e0c0cc0aae619121354" +
		"c173c0eac11abc288b9a83be69a5d6ee86bafd5f1ccc36c85db6503cf42530d3" +
		"24eae1e18b5b366222147a87fa43e6bf6f5f25da0802e3ed1ccab404b556f837" +
		"1a784644ad8a8f91c28091b533f5613cbb99b0b08ead8bf2a71d38e42751ca90" +
		"b1e0e0cc9421b9c4e56fa73b90fd4f3975af1330ec8bf19450880bc38d83ac19" +
		"00d4de876a0728dc13b2bbe1c55c2e9eb14dcdcbdfef8ff0933e72ec3c4d60ae" +
		"8b893254f96e1c83613da6f572accdf02005e44e3d5a580ac43dd523931bf753" +
		"1546934ec60c355555fbc7ad547ad4b27bc940def5885455fcc31178467c3999" +
		"cbcd40d0d192bf24705b42f62bdaba625e8c50f821e9b6a186deea5926ca0db4" +
		"836b8dac3f22ce18e0eb1672bec29178d49e5a15a36a7ad34893b9b35f1e2d1e" +
		"827da27e41e48e65f73c66dfee29a72814acc3a49d4c2a334bf6356aa5491aef" +
		"0b0f113bb4dd9431029b89163878b09e76ef6c94ede081f70ab949e695826c2e" +
		"6fc038a5c2d0a42c26e960157555e41dd759e8bc5f0feb3ca03cf8bb1f53c7b0" +
		"abe05dafaca49f763966d48dc499128cfca0e2c0b8ebfd505d0df94adcfa8dd7" +
		"05bde8e84106f59fa4dd809355f202a4620e41979ec315aecb495e589034faaa" +
		"4f44c46cd7cdd22c48e7a9d94d12c14d35f7c4b3ccb1cdc30add827efc505167" +
		"fcfebb8de3fb79910199203decc077521ae4dec85d8a5922cb19a9821bcf332c" +
		"9784e7c84aa72ef901140011d8aa10bc54dd44f4b680907d15b3aa755cd97f53" +
		"1fd30d81368587090382031bd90a0460a7a492ca9379b0d628e72608d688bb9c" +
		"e33b30a9f457922eac80786951b601d9c91f0d5c36baf7810b2a88a16a5cd535" +
		"4b441162b8cebe6fe50bbacf691b8526c60987641b87813d4bb867c5c57a4e0d" +
		"10d0e2795cc4a1ab325d43ae0dc02f1a64319c2516f0eb0e106e7d5b62c485ad" +
		"fc6a1cc0c9e8417d55469b49bb4f92ce0b2acc0b34245d33738b8689b2190ca2" +
		"4438c01d8552776fb5b0bac994fb60654a6e03ae36215fa00384d3f103eebad7" +
		"cabc8c21f0069c112daf88981556e009609d173d9b41e13ab7bb050f1862ba8f" +
		"3f417b7987a5c19d660b065b07e456ed4de9c5e958676cdaba2642ec93c97cff" +
		"b1d25362a16ded7b6e258c6b5158a050c770245cf0813ccbc06ab581ffa26df2" +
		"35e51fd506b4d33a20007d19311f7c1628875d8588d0cd571d063a9389839fbe" +
		"b18e3d362fe64c55c9116ca5fa756a4c30f1f3f27f1a11cc5f9389b8905cb62d" +
		"15feb4fcff349e035acbc07908cb5f6dc2384a70b8a79518bc0bf9dbbba23106" +
		"b1a7c64fa95535d04be9dee69b187bbfcb9519cb2fae8b2fcd326cb94360521364" +
		"b9422035c25a0e3045fc457399e5f21ac207e62b176210ff49b19d519d192623" +
		"b7efb81dfd6f86f0135b8e7e1330f6b2c3e60d98f91e1ebe57191305e9f9b4e1" +
		"4959417760a2915deaf5772f3773dccd616a0fcf290afaffb8a2796c9de80814" +
		"9f4679"

	expected, _ := hex.DecodeString(expectedHex)
	if len(sig) != len(expected) {
		t.Fatalf("sig length = %d, want %d", len(sig), len(expected))
	}

	for i := range sig {
		if sig[i] != expected[i] {
			t.Errorf("sig[%d] = %02x, want %02x (first mismatch)", i, sig[i], expected[i])
			break
		}
	}
}

// Test sign is deterministic
func TestSignDeterministic(t *testing.T) {
	_, sk := Gen(make([]byte, 32))
	sig1 := Sign(sk, []byte("test"))
	sig2 := Sign(sk, []byte("test"))

	if string(sig1) != string(sig2) {
		t.Error("Sign not deterministic")
	}
}

// Test verify valid signature
func TestVerifyValid(t *testing.T) {
	pk, sk := Gen(make([]byte, 32))
	sig := Sign(sk, []byte("test"))
	if !Verify(pk, []byte("test"), sig) {
		t.Error("Verify returned false for valid signature")
	}
}

// Test verify wrong message
func TestVerifyWrongMessage(t *testing.T) {
	pk, sk := Gen(make([]byte, 32))
	sig := Sign(sk, []byte("test"))
	if Verify(pk, []byte("wrong"), sig) {
		t.Error("Verify returned true for wrong message")
	}
}

// Test verify corrupted signature
func TestVerifyCorruptedSig(t *testing.T) {
	pk, sk := Gen(make([]byte, 32))
	sig := Sign(sk, []byte("test"))
	sig[0] ^= 0xFF
	if Verify(pk, []byte("test"), sig) {
		t.Error("Verify returned true for corrupted signature")
	}
}
