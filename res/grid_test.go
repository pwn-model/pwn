package res

import (
	"testing"

	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

func TestGrid(t *testing.T) {
	g := TreeGrid{Grid: NewGrid[ecs.Entity](20, 10, 1)}

	assert.Equal(t, 20, g.Width())
	assert.Equal(t, 10, g.Height())
	assert.Equal(t, 1, g.CellSize())
	assert.Equal(t, 200, len(g.values))

	assert.True(t, g.values[0].IsZero())

	w := ecs.NewWorld()
	e := w.NewEntity()

	g.Set(0, 5, e)
	assert.False(t, g.values[5].IsZero())

	assert.Equal(t, e, g.Get(0, 5))
}

func TestGridIndexCoords(t *testing.T) {
	g := NewGrid[uint32](20, 10, 1)

	for x := range g.Width() {
		for y := range g.Height() {
			idx := g.Index(x, y)
			assert.Less(t, idx, NoCell)

			cx, cy := g.Coords(idx)
			assert.Equal(t, x, cx)
			assert.Equal(t, y, cy)
		}
	}

	// The index is the position in the grid's own storage.
	g.Set(3, 7, 42)
	assert.Equal(t, uint32(42), g.values[g.Index(3, 7)])
}
