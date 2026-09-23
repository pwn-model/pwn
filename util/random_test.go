package util

import (
	"math"
	"testing"

	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func TestShuffleMatchesJuliaImplementation(t *testing.T) {
	// Hard-coded result of seeding the sibling Julia implementation's RNG
	// with the same seed and shuffling the same slice:
	//   rng = Xoshiro(1); v = collect(1:10); PWNModel.frozen_shuffle!(rng, v)
	// Both implementations must produce this exact permutation for the
	// model runs to select the same trees across languages, regardless of
	// which Julia version is installed (see Shuffle's doc comment).
	expected := []int{2, 7, 4, 5, 1, 6, 10, 9, 3, 8}

	values := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	Shuffle(res.NewXoshiro256pp(1), values)

	assert.Equal(t, expected, values)
}

func TestShuffleEmptyAndSingleton(t *testing.T) {
	empty := []int{}
	assert.NotPanics(t, func() { Shuffle(res.NewXoshiro256pp(1), empty) })

	single := []int{42}
	Shuffle(res.NewXoshiro256pp(1), single)
	assert.Equal(t, []int{42}, single)
}

func TestShuffleIsPermutation(t *testing.T) {
	n := 2500
	values := make([]int, n)
	for i := range values {
		values[i] = i
	}

	Shuffle(res.NewXoshiro256pp(1), values)

	seen := make([]bool, n)
	for _, v := range values {
		assert.False(t, seen[v], "value %d appeared more than once", v)
		seen[v] = true
	}
}

func TestExpFloat64MatchesJuliaImplementation(t *testing.T) {
	// Hard-coded sequence produced by calling the sibling Julia
	// implementation's own randexp directly on the shared Xoshiro256++
	// source with the same seed (not through PWNModel.Rng's low-53-bit
	// uniform override, which randexp never goes through):
	//   rng = Random.Xoshiro(1); [randexp(rng) for _ in 1:5]
	// ExpFloat64 is a deliberate Go port of Julia's own randexp algorithm
	// (see its doc comment), so both implementations must produce this
	// exact sequence from the same seed.
	expected := []float64{
		0.09423776100793935,
		2.1457972199590083,
		1.3231948887670477,
		5.464092249100119,
		2.207171812272818,
	}

	x := res.NewXoshiro256pp(1)
	for _, exp := range expected {
		assert.Equal(t, exp, ExpFloat64(x))
	}
}

func TestExpFloat64IsPositiveAndFinite(t *testing.T) {
	x := res.NewXoshiro256pp(1)
	for range 5000 {
		v := ExpFloat64(x)
		assert.Greater(t, v, 0.0)
		assert.False(t, math.IsInf(v, 0) || math.IsNaN(v))
	}
}
