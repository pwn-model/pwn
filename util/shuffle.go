// Package util holds small, self-contained helpers that don't belong to a
// specific component/resource/system layer.
package util

import (
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
		j := randRange(src, uint64(i+1))
		s[i], s[j] = s[j], s[i]
	}
}

// randRange draws a uniform uint64 in [0, n) via Lemire's multiply-high
// method with rejection sampling near the bias boundary.
func randRange(src rand.Source, n uint64) uint64 {
	hi, lo := bits.Mul64(src.Uint64(), n)
	if lo < n {
		t := -n % n
		for lo < t {
			hi, lo = bits.Mul64(src.Uint64(), n)
		}
	}
	return hi
}
