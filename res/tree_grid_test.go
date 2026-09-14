package res

import (
	"testing"

	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

func TestTreeGrid(t *testing.T) {
	g := NewTreeGrid(20, 10)

	assert.Equal(t, 20, g.Width())
	assert.Equal(t, 10, g.Height())
	assert.Equal(t, 200, len(g.trees))

	assert.True(t, g.trees[0].IsZero())

	w := ecs.NewWorld()
	e := w.NewEntity()

	g.Set(5, 0, e)
	assert.False(t, g.trees[5].IsZero())

	assert.Equal(t, e, g.Get(5, 0))
}
