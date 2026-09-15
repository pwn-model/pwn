package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func TestInitGrids(t *testing.T) {
	app := app.New()
	ws := res.WorldSize{Width: 30, Height: 20, Resolution: 10}
	ecs.AddResource(app.World, &ws)

	s := InitGrids{}
	s.Initialize(app.World)

	trees := ecs.GetResource[res.EntityGrid](app.World)
	assert.Equal(t, ws.Width, trees.Width())
	assert.Equal(t, ws.Height, trees.Height())

	space := ecs.GetResource[res.SpaceGrid](app.World)
	assert.Equal(t, 3, space.Width())  // 30/10
	assert.Equal(t, 2, space.Height()) // 20/10

	q := ecs.NewFilter1[comp.GridCoords](app.World).Query()
	count := 0
	for q.Next() {
		gc := q.Get()
		assert.Equal(t, q.Entity(), space.Get(gc.X, gc.Y))
		count++
	}
	q.Close()

	assert.Equal(t, space.Width()*space.Height(), count)
}

func TestInitGridsPanicsOnSizeNotMultipleOfResolution(t *testing.T) {
	app := app.New()
	ws := res.WorldSize{Width: 25, Height: 12, Resolution: 10}
	ecs.AddResource(app.World, &ws)

	s := InitGrids{}
	assert.Panics(t, func() { s.Initialize(app.World) })
}
