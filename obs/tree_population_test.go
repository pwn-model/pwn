package obs

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/stretchr/testify/assert"
)

func TestTreePopulationObserver(t *testing.T) {
	a := app.New()

	o := TreePopulation{}
	o.Initialize(a.World)

	healthy := ecs.NewMap1[comp.Position](a.World)
	damaged := ecs.NewMap2[comp.Position, comp.Damaged](a.World)
	infected := ecs.NewMap2[comp.Position, comp.Infected](a.World)
	damagedInfected := ecs.NewMap3[comp.Position, comp.Damaged, comp.Infected](a.World)

	healthy.NewEntity(&comp.Position{})
	healthy.NewEntity(&comp.Position{})
	damaged.NewEntity(&comp.Position{}, &comp.Damaged{})
	infected.NewEntity(&comp.Position{}, &comp.Infected{})
	damagedInfected.NewEntity(&comp.Position{}, &comp.Damaged{}, &comp.Infected{})

	assert.Equal(t, []string{"total", "damaged", "infected", "damaged_infected"}, o.Header())
	assert.Equal(t, []float64{5, 1, 1, 1}, o.Values(a.World))
}
