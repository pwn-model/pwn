package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwm-model/pwn/comp"
	"github.com/pwm-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func TestInitTrees(t *testing.T) {
	app := app.New()
	grid := res.NewTreeGrid(100, 50)
	ecs.AddResource(app.World, &grid)

	s := InitTrees{TreeProbability: 0.9}
	s.Initialize(app.World)

	q := ecs.NewFilter1[comp.Position](app.World).Query()
	count := q.Count()
	q.Close()

	assert.Greater(t, count, 4400)
	assert.Less(t, count, 4600)
}
