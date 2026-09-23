// Package util holds small, self-contained helpers that don't belong to a
// specific component/resource/system layer.
package util

import (
	"math"
	"math/bits"
	"math/rand/v2"
)

// Shuffle permutes s in place, using the same algorithm as the sibling
// Julia implementation's frozen_shuffle! (PWNModel.jl/src/util/shuffle.jl):
// a forward Fisher-Yates using Lemire's multiply-high method with rejection
// sampling near the bias boundary (Julia's "Nearly Division Less" ranged
// sampler, see Random.SamplerRangeNDL, https://arxiv.org/abs/1805.10941,
// algorithm 5).
//
// PWNModel.jl deliberately does not call Julia's own Random.shuffle! for
// this: that stdlib function's algorithm has changed at least three times
// across recent Julia releases (confirmed different permutations from the
// same seed on Julia 1.10, 1.12 and 1.13), so it is not a stable target.
// Both implementations instead use this frozen algorithm, so they select
// the same trees from the same seed regardless of which Julia version is
// installed.
func Shuffle[T any](src rand.Source, s []T) {
	n := len(s)
	for i := 1; i < n; i++ {
		j := RandRange(src, uint64(i+1))
		s[i], s[j] = s[j], s[i]
	}
}

// RandRange draws a uniform uint64 in [0, n) via Lemire's multiply-high
// method with rejection sampling near the bias boundary -- the same
// algorithm Shuffle uses (see its doc comment), exported so that any other
// caller needing an index draw that stays bit-identical with the sibling
// Julia implementation's frozen_rand_range (PWNModel.jl/src/util/shuffle.jl)
// can reuse it, rather than reaching for math/rand/v2's own (*Rand).IntN:
// that takes a different, undocumented fast path for power-of-two n (a
// single masked draw, no rejection sampling at all), which silently
// diverges from Julia's algorithm whenever n happens to be a power of two.
func RandRange(src rand.Source, n uint64) uint64 {
	hi, lo := bits.Mul64(src.Uint64(), n)
	if lo < n {
		t := -n % n
		for lo < t {
			hi, lo = bits.Mul64(src.Uint64(), n)
		}
	}
	return hi
}

// ExpFloat64 draws an exponentially-distributed float64 (rate 1) from src,
// using a Go port of Julia's Ziggurat algorithm for randexp (stdlib
// Random.randexp, stdlib/Random/src/normal.jl).
//
// This is deliberately not Go's own math/rand/v2 (*Rand).ExpFloat64: that is
// an independently-implemented, lower-precision Ziggurat variant (32-bit
// table entries, following the original Marsaglia & Tsang paper's reference
// code directly) that consumes source draws differently and produces a
// completely different sequence of values from the same seed. Julia's
// version instead uses 52-bit-precision tables of its own derivation. The
// sibling Julia implementation must draw exponential variates by calling
// randexp directly on the shared Xoshiro256++ source (not through this
// model's usual low-53-bit-biased Rng.rand override, see res/rng.jl), so
// that both implementations consume raw draws identically and produce
// matching sequences from the same seed.
//
// This matches Julia bit-for-bit for the vast majority of draws (confirmed
// over runs of several thousand values), but not with an absolute guarantee:
// roughly one in 256 draws falls into a tail-sampling fallback that computes
// ziggurat_exp_r - log1p(-u), and Go's math.Log1p and Julia's log1p are
// separate library implementations that can disagree by 1 ULP on the rare
// input where one rounds up and the other rounds down. That's an accepted,
// practically-invisible limitation for this stochastic model, not something
// further porting can fix short of also reimplementing log1p bit-for-bit.
func ExpFloat64(src rand.Source) float64 {
	ri := src.Uint64() >> 12
	idx := ri & 0xff
	x := float64(ri) * expWE[idx]
	if ri < expKE[idx] {
		return x
	}
	return expUnlikely(src, idx, x)
}

func expUnlikely(src rand.Source, idx uint64, x float64) float64 {
	if idx == 0 {
		return zigguratExpR - math.Log1p(-expUniform(src))
	}
	if (expFE[idx-1]-expFE[idx])*expUniform(src)+expFE[idx] < math.Exp(-x) {
		return x
	}
	return ExpFloat64(src)
}

// expUniform draws a uniform float64 in [0, 1) using Julia's native
// rand(::Xoshiro) conversion (the high 52 bits of the raw draw), matching
// what Julia's randexp itself calls internally. This is deliberately not
// this model's usual low-53-bit uniform conversion (res.Xoshiro256pp via
// math/rand/v2's Rand.Float64, see res/rng.jl's Rng.rand): ExpFloat64 is a
// self-contained port of Julia's own algorithm, so it must reproduce every
// bit-conversion Julia's randexp performs internally, not this model's
// otherwise-standard uniform convention.
func expUniform(src rand.Source) float64 {
	u := src.Uint64() >> 12
	return math.Float64frombits(0x3ff0000000000000|u) - 1.0
}

// zigguratExpR is the right-hand edge of the Ziggurat's base rectangle for
// the exponential distribution (Julia's Random.ziggurat_exp_r).
const zigguratExpR = 7.6971174701310497140446280481

// expKE, expWE and expFE are Julia's exponential-variate Ziggurat tables
// (Random.ke, Random.we, Random.fe in stdlib/Random/src/normal.jl),
// reproduced bit-for-bit from their raw Float64/UInt64 representations.
var expKE = [256]uint64{
	0x000e290a13924be3, 0x0000000000000000, 0x0009beadebce18bf, 0x000c377ac71f9e08,
	0x000d4ddb99075857, 0x000de893fb8ca23e, 0x000e4a8e87c4328d, 0x000e8dff16ae1cb9,
	0x000ebf2deab58c59, 0x000ee49a6e8b9638, 0x000f0204efd64ee4, 0x000f19bdb8ea3c1b,
	0x000f2d458bbe5bd1, 0x000f3da104b78236, 0x000f4b86d784571f, 0x000f577ad8a7784f,
	0x000f61de83da32ab, 0x000f6afb7843cce7, 0x000f730a57372b44, 0x000f7a37651b0e68,
	0x000f80a5bb6eea52, 0x000f867189d3cb5b, 0x000f8bb1b4f8fbbd, 0x000f9079062292b8,
	0x000f94d70ca8d43a, 0x000f98d8c7dcaa99, 0x000f9c8928abe083, 0x000f9ff175b734a6,
	0x000fa319996bc47d, 0x000fa6085f8e9d07, 0x000fa8c3a62e1991, 0x000fab5084e1f660,
	0x000fadb36c84cccb, 0x000faff041086846, 0x000fb20a6ea22bb9, 0x000fb404fb42cb3c,
	0x000fb5e295158173, 0x000fb7a59e99727a, 0x000fb95038c8789d, 0x000fbae44ba684eb,
	0x000fbc638d822e60, 0x000fbdcf89209ffa, 0x000fbf29a303cfc5, 0x000fc0731df1089c,
	0x000fc1ad1ed6c8b1, 0x000fc2d8b02b5c89, 0x000fc3f6c4d92131, 0x000fc5083ac9ba7d,
	0x000fc60ddd1e9cd6, 0x000fc7086622e825, 0x000fc7f881009f0b, 0x000fc8decb41ac70,
	0x000fc9bbd623d7ec, 0x000fca9027c5b26d, 0x000fcb5c3c319c49, 0x000fcc20864b4449,
	0x000fccdd70a35d40, 0x000fcd935e34bf80, 0x000fce42ab0db8bd, 0x000fceebace7ec01,
	0x000fcf8eb3b0d0e7, 0x000fd02c0a049b60, 0x000fd0c3f59d199c, 0x000fd156b7b5e27e,
	0x000fd1e48d670341, 0x000fd26daff73551, 0x000fd2f2552684be, 0x000fd372af7233c1,
	0x000fd3eeee528f62, 0x000fd4673e73543a, 0x000fd4dbc9e72ff7, 0x000fd54cb856dc2c,
	0x000fd5ba2f2c4119, 0x000fd62451ba02c2, 0x000fd68b415fcff4, 0x000fd6ef1dabc160,
	0x000fd75004790eb6, 0x000fd7ae120c583f, 0x000fd809612dbd09, 0x000fd8620b40effa,
	0x000fd8b8285b78fd, 0x000fd90bcf594b1d, 0x000fd95d15efd425, 0x000fd9ac10bfa70c,
	0x000fd9f8d364df06, 0x000fda437086566b, 0x000fda8bf9e3c9fe, 0x000fdad28062fed5,
	0x000fdb17141bff2c, 0x000fdb59c4648085, 0x000fdb9a9fda83cc, 0x000fdbd9b46e3ed4,
	0x000fdc170f6b5d04, 0x000fdc52bd81a3fb, 0x000fdc8ccacd07ba, 0x000fdcc542dd3902,
	0x000fdcfc30bcb793, 0x000fdd319ef77143, 0x000fdd6597a0f60b, 0x000fdd98245a48a2,
	0x000fddc94e575271, 0x000fddf91e64014f, 0x000fde279ce914ca, 0x000fde54d1f0a06a,
	0x000fde80c52a47cf, 0x000fdeab7def394e, 0x000fded50345eb35, 0x000fdefd5be59fa0,
	0x000fdf248e39b26f, 0x000fdf4aa064b4af, 0x000fdf6f98435894, 0x000fdf937b6f30ba,
	0x000fdfb64f414571, 0x000fdfd818d48262, 0x000fdff8dd07fed8, 0x000fe018a08122c4,
	0x000fe03767adaa59, 0x000fe05536c58a13, 0x000fe07211ccb4c5, 0x000fe08dfc94c532,
	0x000fe0a8fabe8ca1, 0x000fe0c30fbb87a5, 0x000fe0dc3ecf3a5a, 0x000fe0f48b107521,
	0x000fe10bf76a82ef, 0x000fe122869e41ff, 0x000fe1383b4327e1, 0x000fe14d17c83187,
	0x000fe1611e74c023, 0x000fe1745169635a, 0x000fe186b2a09176, 0x000fe19843ef4e07,
	0x000fe1a90705bf63, 0x000fe1b8fd6fb37c, 0x000fe1c828951443, 0x000fe1d689ba4bfd,
	0x000fe1e4220099a4, 0x000fe1f0f26655a0, 0x000fe1fcfbc726d4, 0x000fe2083edc2830,
	0x000fe212bc3bfeb4, 0x000fe21c745adfe3, 0x000fe225678a8895, 0x000fe22d95fa23f4,
	0x000fe234ffb62282, 0x000fe23ba4a800d9, 0x000fe2418495fddc, 0x000fe2469f22bffb,
	0x000fe24af3cce90d, 0x000fe24e81ee9858, 0x000fe25148bcda19, 0x000fe253474703fe,
	0x000fe2547c75fdc6, 0x000fe254e70b754f, 0x000fe25485a0fd1a, 0x000fe25356a71450,
	0x000fe2515864173a, 0x000fe24e88f316f1, 0x000fe24ae64296fa, 0x000fe2466e132f60,
	0x000fe2411df611bd, 0x000fe23af34b6f73, 0x000fe233eb40bf41, 0x000fe22c02cee01b,
	0x000fe22336b81710, 0x000fe2198385e5cc, 0x000fe20ee586b707, 0x000fe20358cb5dfb,
	0x000fe1f6d92465b1, 0x000fe1e9621f2c9e, 0x000fe1daef02c8da, 0x000fe1cb7accb0a6,
	0x000fe1bb002d22c9, 0x000fe1a9798349b8, 0x000fe196e0d9140c, 0x000fe1832fdebc44,
	0x000fe16e5fe5f931, 0x000fe15869dccfcf, 0x000fe1414647fe78, 0x000fe128ed3cf8b2,
	0x000fe10f565b69cf, 0x000fe0f478c633ab, 0x000fe0d84b1bdd9e, 0x000fe0bac36e6688,
	0x000fe09bd73a6b5b, 0x000fe07b7b5d920a, 0x000fe059a40c26d2, 0x000fe03644c5d7f8,
	0x000fe011504979b2, 0x000fdfeab887b95c, 0x000fdfc26e94a447, 0x000fdf986297e305,
	0x000fdf6c83bb8663, 0x000fdf3ec0193eed, 0x000fdf0f04a5d30a, 0x000fdedd3d1aa204,
	0x000fdea953dcfc13, 0x000fde7331e3100d, 0x000fde3abe9626f2, 0x000fddffdfb1dbd5,
	0x000fddc2791ff351, 0x000fdd826cd068c6, 0x000fdd3f9a8d3856, 0x000fdcf9dfc95b0c,
	0x000fdcb1176a55fe, 0x000fdc65198ba50b, 0x000fdc15bb3b2daa, 0x000fdbc2ce2dc4ae,
	0x000fdb6c206aaaca, 0x000fdb117becb4a1, 0x000fdab2a6379bf0, 0x000fda4f5fdfb4e9,
	0x000fd9e76401f3a3, 0x000fd97a67a9ce1f, 0x000fd90819221429, 0x000fd8901f2d4b02,
	0x000fd812182170e1, 0x000fd78d98e23cd3, 0x000fd7022bb3f082, 0x000fd66f4edf96b9,
	0x000fd5d473200305, 0x000fd530f9ccff94, 0x000fd48432b7b351, 0x000fd3cd59a8469e,
	0x000fd30b9368f90a, 0x000fd23dea45f500, 0x000fd16349e2e04a, 0x000fd07a7a3ef98a,
	0x000fcf8219b5df05, 0x000fce7895bcfcde, 0x000fcd5c220ad5e2, 0x000fcc2aadbc17dc,
	0x000fcae1d5e81fbc, 0x000fc97ed4e778f9, 0x000fc7fe6d4d720e, 0x000fc65ccf39c2fc,
	0x000fc4957623cb03, 0x000fc2a2fc826dc7, 0x000fc07ee19b01cd, 0x000fbe213c1cf493,
	0x000fbb8051ac1566, 0x000fb890078d120e, 0x000fb5411a5b9a95, 0x000fb18000547133,
	0x000fad334827f1e2, 0x000fa839276708b9, 0x000fa263b32e37ed, 0x000f9b72d1c52cd1,
	0x000f930a1a281a05, 0x000f889f023d820a, 0x000f7b577d2be5f3, 0x000f69c650c40a8f,
	0x000f51530f0916d8, 0x000f2cb0e3c5933e, 0x000eeefb15d605d8, 0x000e6da6ecf27460,
}

var expWE = [256]float64{
	math.Float64frombits(0x3ce164ec94bf5dc1), math.Float64frombits(0x3c70589d8b5d4119), math.Float64frombits(0x3c7ad6b2495b4d2b),
	math.Float64frombits(0x3c819335a95b8dba), math.Float64frombits(0x3c8522e6e54a2a73), math.Float64frombits(0x3c885090fbc27a80),
	math.Float64frombits(0x3c8b38d1ef79b7cc), math.Float64frombits(0x3c8decd8b76dbd98), math.Float64frombits(0x3c903bf049c65c3c),
	math.Float64frombits(0x3c9170db24d6f670), math.Float64frombits(0x3c92980290da2633), math.Float64frombits(0x3c93b388fe3d6eca),
	math.Float64frombits(0x3c94c515c60bfe21), math.Float64frombits(0x3c95cdf89d024ac3), math.Float64frombits(0x3c96cf40f0a72bbd),
	math.Float64frombits(0x3c97c9cdda17d019), math.Float64frombits(0x3c98be5954d3606f), math.Float64frombits(0x3c99ad80552237d2),
	math.Float64frombits(0x3c9a97c8be5d5203), math.Float64frombits(0x3c9b7da5dddda3c4), math.Float64frombits(0x3c9c5f7bd78c3f89),
	math.Float64frombits(0x3c9d3da24df17c36), math.Float64frombits(0x3c9e186678f1735a), math.Float64frombits(0x3c9ef00ccf5f4faa),
	math.Float64frombits(0x3c9fc4d25d683209), math.Float64frombits(0x3ca04b76ed6a7558), math.Float64frombits(0x3ca0b348479b80fc),
	math.Float64frombits(0x3ca119f38749f5af), math.Float64frombits(0x3ca17f8ceb4bdfa0), math.Float64frombits(0x3ca1e426e93e49e7),
	math.Float64frombits(0x3ca247d26538ff2e), math.Float64frombits(0x3ca2aa9ee123680b), math.Float64frombits(0x3ca30c9aa526da4b),
	math.Float64frombits(0x3ca36dd2e26d8202), math.Float64frombits(0x3ca3ce53d12162a0), math.Float64frombits(0x3ca42e28ca706748),
	math.Float64frombits(0x3ca48d5c5f35e712), math.Float64frombits(0x3ca4ebf86bcd0b93), math.Float64frombits(0x3ca54a0629786f4d),
	math.Float64frombits(0x3ca5a78e3db8befd), math.Float64frombits(0x3ca60498c7dd2ecf), math.Float64frombits(0x3ca6612d6d0c68e0),
	math.Float64frombits(0x3ca6bd5362faa944), math.Float64frombits(0x3ca71911797990bb), math.Float64frombits(0x3ca7746e23077973),
	math.Float64frombits(0x3ca7cf6f7c7e8172), math.Float64frombits(0x3ca82a1b53fed599), math.Float64frombits(0x3ca884772f2be1ec),
	math.Float64frombits(0x3ca8de8850d0c52a), math.Float64frombits(0x3ca93853bdfda244), math.Float64frombits(0x3ca991de42ad1338),
	math.Float64frombits(0x3ca9eb2c75ff03bf), math.Float64frombits(0x3caa4442be14884a), math.Float64frombits(0x3caa9d255396d261),
	math.Float64frombits(0x3caaf5d844f224c9), math.Float64frombits(0x3cab4e5f794c979b), math.Float64frombits(0x3caba6beb33f8f89),
	math.Float64frombits(0x3cabfef99359fe99), math.Float64frombits(0x3cac57139a70d29f), math.Float64frombits(0x3cacaf102bc25adb),
	math.Float64frombits(0x3cad06f28ef0e6fb), math.Float64frombits(0x3cad5ebdf1d86b8d), math.Float64frombits(0x3cadb6756a429057),
	math.Float64frombits(0x3cae0e1bf77c31fe), math.Float64frombits(0x3cae65b483cf1044), math.Float64frombits(0x3caebd41e5e21b62),
	math.Float64frombits(0x3caf14c6e202949f), math.Float64frombits(0x3caf6c462b57feb5), math.Float64frombits(0x3cafc3c26504a9a1),
	math.Float64frombits(0x3cb00d9f119a3cd9), math.Float64frombits(0x3cb0395df60db162), math.Float64frombits(0x3cb0651f1c7276f8),
	math.Float64frombits(0x3cb090e3bb4b0072), math.Float64frombits(0x3cb0bcad03710137), math.Float64frombits(0x3cb0e87c207a2f66),
	math.Float64frombits(0x3cb114523917ac15), math.Float64frombits(0x3cb140306f707dbe), math.Float64frombits(0x3cb16c17e1777ffb),
	math.Float64frombits(0x3cb19809a93d2396), math.Float64frombits(0x3cb1c406dd3d5283), math.Float64frombits(0x3cb1f01090a9c4e2),
	math.Float64frombits(0x3cb21c27d3b10e05), math.Float64frombits(0x3cb2484db3c2a329), math.Float64frombits(0x3cb274833bd0189f),
	math.Float64frombits(0x3cb2a0c9748bcdaa), math.Float64frombits(0x3cb2cd2164a53b5d), math.Float64frombits(0x3cb2f98c11031721),
	math.Float64frombits(0x3cb3260a7cfb7611), math.Float64frombits(0x3cb3529daa8a1ba1), math.Float64frombits(0x3cb37f469a851af0),
	math.Float64frombits(0x3cb3ac064ccfeffc), math.Float64frombits(0x3cb3d8ddc08d336d), math.Float64frombits(0x3cb405cdf44f09c4),
	math.Float64frombits(0x3cb432d7e6466cd0), math.Float64frombits(0x3cb45ffc94716ca7), math.Float64frombits(0x3cb48d3cfcc883c4),
	math.Float64frombits(0x3cb4ba9a1d6b18a4), math.Float64frombits(0x3cb4e814f4cb45ea), math.Float64frombits(0x3cb515ae81d900fb),
	math.Float64frombits(0x3cb54367c42cb5f8), math.Float64frombits(0x3cb57141bc316f27), math.Float64frombits(0x3cb59f3d6b4e9cf9),
	math.Float64frombits(0x3cb5cd5bd4119335), math.Float64frombits(0x3cb5fb9dfa56cf26), math.Float64frombits(0x3cb62a04e3731a2e),
	math.Float64frombits(0x3cb65891965c9b8c), math.Float64frombits(0x3cb687451bd3ebee), math.Float64frombits(0x3cb6b6207e8d3cdf),
	math.Float64frombits(0x3cb6e524cb59a608), math.Float64frombits(0x3cb714531150a9fb), math.Float64frombits(0x3cb743ac61fa041c),
	math.Float64frombits(0x3cb77331d177d130), math.Float64frombits(0x3cb7a2e476b1240a), math.Float64frombits(0x3cb7d2c56b7d17f7),
	math.Float64frombits(0x3cb802d5ccce7277), math.Float64frombits(0x3cb83316badfe62a), math.Float64frombits(0x3cb86389596108e7),
	math.Float64frombits(0x3cb8942ecfa40f54), math.Float64frombits(0x3cb8c50848cc6094), math.Float64frombits(0x3cb8f616f3fe1513),
	math.Float64frombits(0x3cb9275c048e73e1), math.Float64frombits(0x3cb958d8b235828a), math.Float64frombits(0x3cb98a8e3940bbf4),
	math.Float64frombits(0x3cb9bc7ddac7035d), math.Float64frombits(0x3cb9eea8dcdde951), math.Float64frombits(0x3cba21108ad0592d),
	math.Float64frombits(0x3cba53b63556c690), math.Float64frombits(0x3cba869b32d0f30f), math.Float64frombits(0x3cbab9c0df81657a),
	math.Float64frombits(0x3cbaed289dcaacff), math.Float64frombits(0x3cbb20d3d66e8bb5), math.Float64frombits(0x3cbb54c3f8cf2542),
	math.Float64frombits(0x3cbb88fa7b324fb6), math.Float64frombits(0x3cbbbd78db072610), math.Float64frombits(0x3cbbf2409d2dfd85),
	math.Float64frombits(0x3cbc27534e42e02d), math.Float64frombits(0x3cbc5cb282eab1a4), math.Float64frombits(0x3cbc925fd82323fb),
	math.Float64frombits(0x3cbcc85cf395a56c), math.Float64frombits(0x3cbcfeab83ed7180), math.Float64frombits(0x3cbd354d4130f2ad),
	math.Float64frombits(0x3cbd6c43ed1ea3fe), math.Float64frombits(0x3cbda391538da50a), math.Float64frombits(0x3cbddb374ad2357f),
	math.Float64frombits(0x3cbe1337b426509b), math.Float64frombits(0x3cbe4b947c16a452), math.Float64frombits(0x3cbe844f9af4237f),
	math.Float64frombits(0x3cbebd6b154a7678), math.Float64frombits(0x3cbef6e8fc5b9168), math.Float64frombits(0x3cbf30cb6ea0bc7f),
	math.Float64frombits(0x3cbf6b1498515ed0), math.Float64frombits(0x3cbfa5c6b3efe1e5), math.Float64frombits(0x3cbfe0e40add09d8),
	math.Float64frombits(0x3cc00e377af911d4), math.Float64frombits(0x3cc02c34ef11391b), math.Float64frombits(0x3cc04a6b9e9224a3),
	math.Float64frombits(0x3cc068dccf1126db), math.Float64frombits(0x3cc08789cf3aad0f), math.Float64frombits(0x3cc0a673f733c819),
	math.Float64frombits(0x3cc0c59ca900946f), math.Float64frombits(0x3cc0e50550efcfb7), math.Float64frombits(0x3cc104af660befce),
	math.Float64frombits(0x3cc1249c6a92154a), math.Float64frombits(0x3cc144cdec6f3a2b), math.Float64frombits(0x3cc1654585c404c1),
	math.Float64frombits(0x3cc18604dd6fae9e), math.Float64frombits(0x3cc1a70da7a27820), math.Float64frombits(0x3cc1c861a6782a5a),
	math.Float64frombits(0x3cc1ea02aa9b3370), math.Float64frombits(0x3cc20bf293f0f4a2), math.Float64frombits(0x3cc22e33524fe550),
	math.Float64frombits(0x3cc250c6e6403bba), math.Float64frombits(0x3cc273af61c7daa6), math.Float64frombits(0x3cc296eee942532b),
	math.Float64frombits(0x3cc2ba87b445db51), math.Float64frombits(0x3cc2de7c0e962d70), math.Float64frombits(0x3cc302ce59265965),
	math.Float64frombits(0x3cc327810b2aa7d0), math.Float64frombits(0x3cc34c96b33bc965), math.Float64frombits(0x3cc37211f88ca856),
	math.Float64frombits(0x3cc397f59c345143), math.Float64frombits(0x3cc3be447a8d8b83), math.Float64frombits(0x3cc3e5018cadded0),
	math.Float64frombits(0x3cc40c2fe9f5eead), math.Float64frombits(0x3cc433d2c9bd42f8), math.Float64frombits(0x3cc45bed851bc92c),
	math.Float64frombits(0x3cc4848398d39432), math.Float64frombits(0x3cc4ad98a75da14c), math.Float64frombits(0x3cc4d7307b1cb127),
	math.Float64frombits(0x3cc5014f08b99508), math.Float64frombits(0x3cc52bf871acaab2), math.Float64frombits(0x3cc5573106f8a75a),
	math.Float64frombits(0x3cc582fd4c1b4461), math.Float64frombits(0x3cc5af61fa38e107), math.Float64frombits(0x3cc5dc640388bd9e),
	math.Float64frombits(0x3cc60a0897081879), math.Float64frombits(0x3cc63855247b2e94), math.Float64frombits(0x3cc6674f60c3f432),
	math.Float64frombits(0x3cc696fd4a9748ee), math.Float64frombits(0x3cc6c7652f9a7b1e), math.Float64frombits(0x3cc6f88db1f42507),
	math.Float64frombits(0x3cc72a7dce5cd218), math.Float64frombits(0x3cc75d3ce2bd71c3), math.Float64frombits(0x3cc790d2b56b71f9),
	math.Float64frombits(0x3cc7c5477d1476d3), math.Float64frombits(0x3cc7faa3e96e1412), math.Float64frombits(0x3cc830f12cc0bec3),
	math.Float64frombits(0x3cc8683906687342), math.Float64frombits(0x3cc8a085ce695bab), math.Float64frombits(0x3cc8d9e2823b3695),
	math.Float64frombits(0x3cc9145ad2f37544), math.Float64frombits(0x3cc94ffb34fc2a0e), math.Float64frombits(0x3cc98cd0f18d1ad8),
	math.Float64frombits(0x3cc9caea3a24d9ea), math.Float64frombits(0x3cca0a563e49f178), math.Float64frombits(0x3cca4b2543e84c3b),
	math.Float64frombits(0x3cca8d68c2ad86ea), math.Float64frombits(0x3ccad13382d845c4), math.Float64frombits(0x3ccb1699c003b60a),
	math.Float64frombits(0x3ccb5db15091ea0f), math.Float64frombits(0x3ccba691d276da5e), math.Float64frombits(0x3ccbf154de4bef77),
	math.Float64frombits(0x3ccc3e1641c2e0a7), math.Float64frombits(0x3ccc8cf442c8c8f4), math.Float64frombits(0x3cccde0fecf2a97f),
	math.Float64frombits(0x3ccd318d6b2738c5), math.Float64frombits(0x3ccd87946fec3bec), math.Float64frombits(0x3ccde050af4ef19f),
	math.Float64frombits(0x3cce3bf26e190960), math.Float64frombits(0x3cce9aaf2af383c1), math.Float64frombits(0x3ccefcc26750ea4a),
	math.Float64frombits(0x3ccf626e9791f7a7), math.Float64frombits(0x3ccfcbfe43f6c6e5), math.Float64frombits(0x3cd01ce2b362ec2e),
	math.Float64frombits(0x3cd056118bf58eef), math.Float64frombits(0x3cd091c1cdcba54e), math.Float64frombits(0x3cd0d031785d48a0),
	math.Float64frombits(0x3cd111a8034392a6), math.Float64frombits(0x3cd156786775442a), math.Float64frombits(0x3cd19f03bcb3c2d6),
	math.Float64frombits(0x3cd1ebbca0c9fa7c), math.Float64frombits(0x3cd23d2bb659919f), math.Float64frombits(0x3cd293f5ae49aaa5),
	math.Float64frombits(0x3cd2f0e38a4411f0), math.Float64frombits(0x3cd354ee27ccf75e), math.Float64frombits(0x3cd3c14ec7c8b861),
	math.Float64frombits(0x3cd4379766e41362), math.Float64frombits(0x3cd4b9d7cd4751d1), math.Float64frombits(0x3cd54ad83ccf73f6),
	math.Float64frombits(0x3cd5ee7ae17313d2), math.Float64frombits(0x3cd6aa676d4bbf72), math.Float64frombits(0x3cd78750d6eac62f),
	math.Float64frombits(0x3cd8939fe6f2ed19), math.Float64frombits(0x3cd9e9dc0d487b85), math.Float64frombits(0x3cdbc39e51da71fc),
	math.Float64frombits(0x3cdec9d9297ebb83),
}

var expFE = [256]float64{
	math.Float64frombits(0x3ff0000000000000), math.Float64frombits(0x3fee0545e5881137), math.Float64frombits(0x3fecd0a65081fff1),
	math.Float64frombits(0x3febe5007beb7b27), math.Float64frombits(0x3feb210f0ee67f2a), math.Float64frombits(0x3fea76baa562fae7),
	math.Float64frombits(0x3fe9de9715556d9b), math.Float64frombits(0x3fe95431c455aa39), math.Float64frombits(0x3fe8d4a376d3d22f),
	math.Float64frombits(0x3fe85de87806c5b8), math.Float64frombits(0x3fe7ee8a2d243126), math.Float64frombits(0x3fe7856e9b09d47e),
	math.Float64frombits(0x3fe721bb5ba94b63), math.Float64frombits(0x3fe6c2c3498418c6), math.Float64frombits(0x3fe667fa6d4f5c06),
	math.Float64frombits(0x3fe610edc1a7af66), math.Float64frombits(0x3fe5bd3d694cac75), math.Float64frombits(0x3fe56c9882da8773),
	math.Float64frombits(0x3fe51eba1578899a), math.Float64frombits(0x3fe4d366c151f8af), math.Float64frombits(0x3fe48a6afb8ee069),
	math.Float64frombits(0x3fe44399afa8e125), math.Float64frombits(0x3fe3fecb2bb18b80), math.Float64frombits(0x3fe3bbdc44e1d114),
	math.Float64frombits(0x3fe37aada708ddd9), math.Float64frombits(0x3fe33b23450e6318), math.Float64frombits(0x3fe2fd23e345da5e),
	math.Float64frombits(0x3fe2c098b61f4f24), math.Float64frombits(0x3fe2856d111132bd), math.Float64frombits(0x3fe24b8e228c50a3),
	math.Float64frombits(0x3fe212eaba813ec8), math.Float64frombits(0x3fe1db7319877b89), math.Float64frombits(0x3fe1a518c71e3b25),
	math.Float64frombits(0x3fe16fce6dce6fee), math.Float64frombits(0x3fe13b87bc33169c), math.Float64frombits(0x3fe108394a1cc38d),
	math.Float64frombits(0x3fe0d5d8812b1e2b), math.Float64frombits(0x3fe0a45b8854d02a), math.Float64frombits(0x3fe073b931ee3b7d),
	math.Float64frombits(0x3fe043e8ebd26548), math.Float64frombits(0x3fe014e2b160f324), math.Float64frombits(0x3fdfcd3dfe214576),
	math.Float64frombits(0x3fdf722d8ebfc5fa), math.Float64frombits(0x3fdf1886d1eb424d), math.Float64frombits(0x3fdec03d4b969d90),
	math.Float64frombits(0x3fde6945367dd351), math.Float64frombits(0x3fde139375e137fc), math.Float64frombits(0x3fddbf1d88a7210c),
	math.Float64frombits(0x3fdd6bd97db9ed7a), math.Float64frombits(0x3fdd19bde97e1a0b), math.Float64frombits(0x3fdcc8c1dc40e092),
	math.Float64frombits(0x3fdc78dcd983fb60), math.Float64frombits(0x3fdc2a06d00ea583), math.Float64frombits(0x3fdbdc3812aeeeb5),
	math.Float64frombits(0x3fdb8f6951990b88), math.Float64frombits(0x3fdb43939454806f), math.Float64frombits(0x3fdaf8b03428ef5f),
	math.Float64frombits(0x3fdaaeb8d6fdf6e5), math.Float64frombits(0x3fda65a76aa30140), math.Float64frombits(0x3fda1d76207521f4),
	math.Float64frombits(0x3fd9d61f695a3792), math.Float64frombits(0x3fd98f9df2097ba8), math.Float64frombits(0x3fd949ec9f9a8110),
	math.Float64frombits(0x3fd905068c545d04), math.Float64frombits(0x3fd8c0e704b75d39), math.Float64frombits(0x3fd87d8984bc3f8c),
	math.Float64frombits(0x3fd83ae9b5446138), math.Float64frombits(0x3fd7f90369b6ce59), math.Float64frombits(0x3fd7b7d29dc6801e),
	math.Float64frombits(0x3fd77753735e72e3), math.Float64frombits(0x3fd7378230b08dea), math.Float64frombits(0x3fd6f85b3e649e9d),
	math.Float64frombits(0x3fd6b9db25e4e99c), math.Float64frombits(0x3fd67bfe8fc60d9f), math.Float64frombits(0x3fd63ec2424827e4),
	math.Float64frombits(0x3fd602231fef5876), math.Float64frombits(0x3fd5c61e2631ee6c), math.Float64frombits(0x3fd58ab06c3aa9ef),
	math.Float64frombits(0x3fd54fd721bda3e7), math.Float64frombits(0x3fd5158f8dde89f5), math.Float64frombits(0x3fd4dbd70e26f91d),
	math.Float64frombits(0x3fd4a2ab158bdad3), math.Float64frombits(0x3fd46a092b80beef), math.Float64frombits(0x3fd431eeeb1841e2),
	math.Float64frombits(0x3fd3fa5a0230a14e), math.Float64frombits(0x3fd3c34830abb285), math.Float64frombits(0x3fd38cb747b17def),
	math.Float64frombits(0x3fd356a528fcd0dd), math.Float64frombits(0x3fd3210fc6312435), math.Float64frombits(0x3fd2ebf520394270),
	math.Float64frombits(0x3fd2b75346ae2262), math.Float64frombits(0x3fd2832857457629), math.Float64frombits(0x3fd24f727d4776fd),
	math.Float64frombits(0x3fd21c2ff10b7eff), math.Float64frombits(0x3fd1e95ef77b09db), math.Float64frombits(0x3fd1b6fde19abc5a),
	math.Float64frombits(0x3fd1850b0c191982), math.Float64frombits(0x3fd15384dee291ef), math.Float64frombits(0x3fd12269ccba9fba),
	math.Float64frombits(0x3fd0f1b852d9a66c), math.Float64frombits(0x3fd0c16ef88f5333), math.Float64frombits(0x3fd0918c4ee93e13),
	math.Float64frombits(0x3fd0620ef05d90d2), math.Float64frombits(0x3fd032f580797c2c), math.Float64frombits(0x3fd0043eab93476a),
	math.Float64frombits(0x3fcfabd24cff9354), math.Float64frombits(0x3fcf4fe75c963e7e), math.Float64frombits(0x3fcef4ba0fe8e09b),
	math.Float64frombits(0x3fce9a48005940f2), math.Float64frombits(0x3fce408ed62f83a7), math.Float64frombits(0x3fcde78c48224f39),
	math.Float64frombits(0x3fcd8f3e1ae3eeb8), math.Float64frombits(0x3fcd37a220b431fd), math.Float64frombits(0x3fcce0b638f6d09f),
	math.Float64frombits(0x3fcc8a784fce1802), math.Float64frombits(0x3fcc34e65db9afee), math.Float64frombits(0x3fcbdffe67394435),
	math.Float64frombits(0x3fcb8bbe7c72e4a5), math.Float64frombits(0x3fcb3824b8dcef3e), math.Float64frombits(0x3fcae52f42eb5b0b),
	math.Float64frombits(0x3fca92dc4bc03c49), math.Float64frombits(0x3fca412a0edf5cbc), math.Float64frombits(0x3fc9f016d1e4c512),
	math.Float64frombits(0x3fc99fa0e43e1623), math.Float64frombits(0x3fc94fc69ee692a1), math.Float64frombits(0x3fc900866425bb79),
	math.Float64frombits(0x3fc8b1de9f5062d5), math.Float64frombits(0x3fc863cdc48c1af9), math.Float64frombits(0x3fc816525094e7e6),
	math.Float64frombits(0x3fc7c96ac8851bae), math.Float64frombits(0x3fc77d15b99f46fe), math.Float64frombits(0x3fc73151b91a2839),
	math.Float64frombits(0x3fc6e61d63ee84ea), math.Float64frombits(0x3fc69b775ea6da28), math.Float64frombits(0x3fc6515e5530d1ac),
	math.Float64frombits(0x3fc607d0fab06a31), math.Float64frombits(0x3fc5bece0954c2b6), math.Float64frombits(0x3fc57654422e78f5),
	math.Float64frombits(0x3fc52e626d078c49), math.Float64frombits(0x3fc4e6f7583cb6fa), math.Float64frombits(0x3fc4a011d8983096),
	math.Float64frombits(0x3fc459b0c92dccc6), math.Float64frombits(0x3fc413d30b386a9a), math.Float64frombits(0x3fc3ce7785f8a905),
	math.Float64frombits(0x3fc3899d2694d5c9), math.Float64frombits(0x3fc34542dffa0caf), math.Float64frombits(0x3fc30167aabe7d6e),
	math.Float64frombits(0x3fc2be0a8504cf34), math.Float64frombits(0x3fc27b2a72609940), math.Float64frombits(0x3fc238c67bbbe878),
	math.Float64frombits(0x3fc1f6ddaf3dca65), math.Float64frombits(0x3fc1b56f2031d666), math.Float64frombits(0x3fc17479e6f0ae78),
	math.Float64frombits(0x3fc133fd20c9712f), math.Float64frombits(0x3fc0f3f7efec1720), math.Float64frombits(0x3fc0b4697b54b62f),
	math.Float64frombits(0x3fc07550eeb7a5be), math.Float64frombits(0x3fc036ad7a6e7f04), math.Float64frombits(0x3fbff0fca6cbea8d),
	math.Float64frombits(0x3fbf758566190414), math.Float64frombits(0x3fbefaf3ae83c33c), math.Float64frombits(0x3fbe8146048eb9cc),
	math.Float64frombits(0x3fbe087af561bafb), math.Float64frombits(0x3fbd909116ad9398), math.Float64frombits(0x3fbd198706914dd7),
	math.Float64frombits(0x3fbca35b6b80fd57), math.Float64frombits(0x3fbc2e0cf42e10af), math.Float64frombits(0x3fbbb99a5771268f),
	math.Float64frombits(0x3fbb460254356548), math.Float64frombits(0x3fbad343b1655465), math.Float64frombits(0x3fba615d3dd938b7),
	math.Float64frombits(0x3fb9f04dd046f428), math.Float64frombits(0x3fb9801447336b70), math.Float64frombits(0x3fb910af88e574b9),
	math.Float64frombits(0x3fb8a21e835a533b), math.Float64frombits(0x3fb834602c3bc4ba), math.Float64frombits(0x3fb7c77380d7a6f3),
	math.Float64frombits(0x3fb75b5786193c1e), math.Float64frombits(0x3fb6f00b488416b6), math.Float64frombits(0x3fb6858ddc30b620),
	math.Float64frombits(0x3fb61bde5ccadef7), math.Float64frombits(0x3fb5b2fbed91bb3e), math.Float64frombits(0x3fb54ae5b959d036),
	math.Float64frombits(0x3fb4e39af290d929), math.Float64frombits(0x3fb47d1ad343985c), math.Float64frombits(0x3fb417649d25b10e),
	math.Float64frombits(0x3fb3b277999b9f9e), math.Float64frombits(0x3fb34e5319c6e718), math.Float64frombits(0x3fb2eaf676948dd1),
	math.Float64frombits(0x3fb2886110ce0570), math.Float64frombits(0x3fb22692512c9d8c), math.Float64frombits(0x3fb1c589a86fa340),
	math.Float64frombits(0x3fb165468f755392), math.Float64frombits(0x3fb105c88756ca50), math.Float64frombits(0x3fb0a70f19871b3b),
	math.Float64frombits(0x3fb04919d7f5c817), math.Float64frombits(0x3fafd7d0ba699676), math.Float64frombits(0x3faf1ef49944e834),
	math.Float64frombits(0x3fae679ea52eb2e5), math.Float64frombits(0x3fadb1ce49315810), math.Float64frombits(0x3facfd83031e794a),
	math.Float64frombits(0x3fac4abc640721e9), math.Float64frombits(0x3fab997a10bed985), math.Float64frombits(0x3faae9bbc26a8084),
	math.Float64frombits(0x3faa3b81471bf138), math.Float64frombits(0x3fa98eca827b7c4c), math.Float64frombits(0x3fa8e3976e80776d),
	math.Float64frombits(0x3fa839e81c3a396b), math.Float64frombits(0x3fa791bcb4ab089e), math.Float64frombits(0x3fa6eb1579b6af52),
	math.Float64frombits(0x3fa645f2c726a041), math.Float64frombits(0x3fa5a25513c5d2ca), math.Float64frombits(0x3fa5003cf296c5eb),
	math.Float64frombits(0x3fa45fab14266b19), math.Float64frombits(0x3fa3c0a047ff18ff), math.Float64frombits(0x3fa3231d7e3f14ae),
	math.Float64frombits(0x3fa28723c956c00c), math.Float64frombits(0x3fa1ecb45ff312d4), math.Float64frombits(0x3fa153d09f19b3a1),
	math.Float64frombits(0x3fa0bc7a0c7cd651), math.Float64frombits(0x3fa026b2590dfaee), math.Float64frombits(0x3f9f24f6c7af9890),
	math.Float64frombits(0x3f9dffae7a517468), math.Float64frombits(0x3f9cdd9054331b0c), math.Float64frombits(0x3f9bbea150fa5870),
	math.Float64frombits(0x3f9aa2e6e6924e9b), math.Float64frombits(0x3f998a670f132a48), math.Float64frombits(0x3f98752853ec9967),
	math.Float64frombits(0x3f976331da87fc96), math.Float64frombits(0x3f96548b72a24077), math.Float64frombits(0x3f95493da6ab0251),
	math.Float64frombits(0x3f944151ce87f0be), math.Float64frombits(0x3f933cd225315d84), math.Float64frombits(0x3f923bc9e1b93a32),
	math.Float64frombits(0x3f913e4554725f5f), math.Float64frombits(0x3f904452091e02f0), math.Float64frombits(0x3f8e9bfdde89c7ce),
	math.Float64frombits(0x3f8cb6b9146e2757), math.Float64frombits(0x3f8ad8fa5542c92d), math.Float64frombits(0x3f8902ea688fa7bd),
	math.Float64frombits(0x3f8734b6e6aa74f5), math.Float64frombits(0x3f856e930be416cb), math.Float64frombits(0x3f83b0b8c1516f62),
	math.Float64frombits(0x3f81fb69edb37671), math.Float64frombits(0x3f804ef2295fd7f9), math.Float64frombits(0x3f7d5751fa745dc5),
	math.Float64frombits(0x3f7a23e9d4974836), math.Float64frombits(0x3f77049f37ec3620), math.Float64frombits(0x3f73fa97cee322fd),
	math.Float64frombits(0x3f71073d69574043), math.Float64frombits(0x3f6c58b381cd4b11), math.Float64frombits(0x3f66d888f3a1feff),
	math.Float64frombits(0x3f61946ba8e1a324), math.Float64frombits(0x3f592bb5540c3e25), math.Float64frombits(0x3f4fb20af78dfcb9),
	math.Float64frombits(0x3f3dc31c329f0b4b),
}
