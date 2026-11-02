package sys

import (
	"testing"

	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func newMaxFieldNeighborSetup(seed uint64) (*BeetleMovement, res.Grid[float64], *res.Xoshiro256pp) {
	s := &BeetleMovement{width: 10, height: 10}
	field := res.NewGrid[float64](10, 10, 10)
	return s, field, res.NewXoshiro256pp(seed)
}

func TestMaxFieldNeighborUniqueMax(t *testing.T) {
	s, field, src := newMaxFieldNeighborSetup(1)
	field.Fill(0.5)
	field.Set(6, 4, 0.9)

	ref := res.NewXoshiro256pp(1)

	x, y := s.maxFieldNeighbor(src, &field, 5, 5)
	assert.Equal(t, 6, x)
	assert.Equal(t, 4, y)

	// Without a tie, no random number is drawn.
	assert.Equal(t, ref.Uint64(), src.Uint64())
}

func TestMaxFieldNeighborTieOnlyPicksCandidates(t *testing.T) {
	s, field, src := newMaxFieldNeighborSetup(1)
	field.Fill(0.5)
	field.Set(4, 4, 0.9)
	field.Set(6, 6, 0.9)

	counts := map[[2]int]int{}
	for range 1000 {
		x, y := s.maxFieldNeighbor(src, &field, 5, 5)
		counts[[2]int{x, y}]++
	}

	assert.Len(t, counts, 2)
	assert.InDelta(t, 500, counts[[2]int{4, 4}], 75)
	assert.InDelta(t, 500, counts[[2]int{6, 6}], 75)
}

// On a flat field, all neighbors are equally likely, without bias towards
// the first neighborOffsets.
func TestMaxFieldNeighborFlatFieldUniform(t *testing.T) {
	s, field, src := newMaxFieldNeighborSetup(1)
	field.Fill(0.5)

	n := 8000
	counts := map[[2]int]int{}
	for range n {
		x, y := s.maxFieldNeighbor(src, &field, 5, 5)
		counts[[2]int{x - 5, y - 5}]++
	}

	assert.Len(t, counts, len(neighborOffsets))
	for _, o := range neighborOffsets {
		assert.InDelta(t, n/len(neighborOffsets), counts[o], 120, "offset %v", o)
	}
}

// At the world's corner, only in-bounds neighbors are candidates.
func TestMaxFieldNeighborFlatFieldCorner(t *testing.T) {
	s, field, src := newMaxFieldNeighborSetup(1)
	field.Fill(0.5)

	counts := map[[2]int]int{}
	for range 300 {
		x, y := s.maxFieldNeighbor(src, &field, 0, 0)
		counts[[2]int{x, y}]++
	}

	assert.Len(t, counts, 3)
	for _, c := range [][2]int{{1, 0}, {0, 1}, {1, 1}} {
		assert.Positive(t, counts[c], "cell %v", c)
	}
}
