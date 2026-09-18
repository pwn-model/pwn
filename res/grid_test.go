package res

import (
	"testing"

	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

func TestGrid(t *testing.T) {
	g := TreeGrid{Grid: NewGrid[ecs.Entity](20, 10)}

	assert.Equal(t, 20, g.Width())
	assert.Equal(t, 10, g.Height())
	assert.Equal(t, 200, len(g.values))

	assert.True(t, g.values[0].IsZero())

	w := ecs.NewWorld()
	e := w.NewEntity()

	g.Set(0, 5, e)
	assert.False(t, g.values[5].IsZero())

	assert.Equal(t, e, g.Get(0, 5))
}
