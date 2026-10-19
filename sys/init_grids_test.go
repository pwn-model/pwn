package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func TestInitGrids(t *testing.T) {
	app := app.New()
	ws := res.NewWorldSize(300, 200, 10)
	ecs.AddResource(app.World, &ws)

	s := InitGrids{}
	s.Initialize(app.World)

	trees := ecs.GetResource[res.TreeGrid](app.World)
	assert.Equal(t, ws.Width(), trees.Width())
	assert.Equal(t, ws.Height(), trees.Height())
}
