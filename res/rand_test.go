package res

import (
	"math/rand/v2"
	"testing"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/stretchr/testify/assert"
)

// TestRandSequence checks that resource.Rand produces a fixed, hard-coded
// sequence of values for a given seed. This is the same seed and sequence
// hard-coded in the sibling Julia implementation's Rng test, so that both
// implementations are confirmed to stay reproducible across languages.
func TestRandSequence(t *testing.T) {
	r := rand.New(resource.Rand{Source: rand.NewPCG(0, 1)})

	expected := []float64{
		0.47114790869927514,
		0.7197903592279431,
		0.8082559527889128,
		0.7087981878729903,
		0.1539910046124079,
	}

	for _, exp := range expected {
		assert.Equal(t, exp, r.Float64())
	}
}
