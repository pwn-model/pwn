package res

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestXoshiro256ppSequence checks that Xoshiro256pp produces the same
// hard-coded sequence as Julia's Random.Xoshiro for the same seed:
//
//	rng = Random.Xoshiro(1); [rand(rng, UInt64) for _ in 1:5]
func TestXoshiro256ppSequence(t *testing.T) {
	x := NewXoshiro256pp(1)

	expected := []uint64{
		1353370364516103630,
		6442368377782521601,
		12891076985935696735,
		11589438875633714339,
		16877461176182319662,
	}

	for _, exp := range expected {
		assert.Equal(t, exp, x.Uint64())
	}
}
