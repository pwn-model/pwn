// Package util holds small, self-contained helpers that don't belong to a
// specific component/resource/system layer.
package util

import "math/rand/v2"

// Shuffle permutes s in place, using the same algorithm as Julia's
// Random.shuffle! (stdlib Random.jl) for an Xoshiro-backed RNG: a forward
// Fisher-Yates variant that draws each swap index from the high 52 bits of
// a raw Uint64 draw (matching Julia's UInt52Raw, which right-shifts a raw
// draw by 12 bits rather than masking its low bits), masks that down to a
// bitmask sized to the current index, and rejects out-of-range results,
// growing the mask as needed. This differs from math/rand/v2's
// Lemire-multiply-based [rand.Rand.Shuffle].
//
// This must be kept in sync with Julia's shuffle!, which the sibling Julia
// implementation relies on for tree selection (see
// PWNModel.jl/src/sys/random_infection.jl), so that both implementations
// select the same trees from the same seed.
func Shuffle[T any](src rand.Source, s []T) {
	n := len(s)
	if n == 0 {
		return
	}

	mask := uint64(3)
	for i := 1; i < n; i++ {
		sup := uint64(i)

		var j uint64
		for {
			j = (src.Uint64() >> 12) & mask
			if j <= sup {
				break
			}
		}

		s[i], s[j] = s[j], s[i]
		if uint64(i) == mask {
			mask = 2*mask + 1
		}
	}
}
