package util

import (
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
