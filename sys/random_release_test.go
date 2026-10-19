package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func setupRandomReleaseWorld(t *testing.T) *ecs.World {
	t.Helper()

	a := app.New()

	// 20x20 cells, one tree per cell.
	ws := res.NewWorldSize(200, 200, 10)
	ecs.AddResource(a.World, &ws)

	gs := InitGrids{}
	gs.Initialize(a.World)

	ts := InitTrees{Km2Probability: 1.0, TreeProbability: 1.0}
	ts.Initialize(a.World)

	return a.World
}

func TestRandomRelease(t *testing.T) {
	world := setupRandomReleaseWorld(t)

	s := RandomRelease{TickOfInfection: 3, NumTrees: 10}
	s.Initialize(world)

	time := res.Time{}
	ecs.AddResource(world, &time)
	infected := ecs.NewFilter1[comp.Infected](world)

	// Ticks before TickOfInfection must not infect anything.
	for time.Tick = 0; time.Tick < 3; time.Tick++ {
		s.Update(world)
	}
	q := infected.Query()
	assert.Equal(t, 0, q.Count())
	q.Close()

	// At TickOfInfection, exactly NumTrees trees get infected.
	s.Update(world)

	infQuery := infected.Query()
	count := 0
	for infQuery.Next() {
		inf := infQuery.Get()
		assert.Equal(t, 3, inf.InfectionTick)
		count++
	}
	infQuery.Close()
	assert.Equal(t, 10, count)

	// Later ticks must not infect further trees.
	time.Tick = 4
	s.Update(world)

	q = infected.Query()
	assert.Equal(t, 10, q.Count())
	q.Close()
}

func TestRandomReleaseSkipsIneligibleTrees(t *testing.T) {
	world := setupRandomReleaseWorld(t)

	// Mark all trees as already infected, except one.
	infectMapper := ecs.NewMap1[comp.Infected](world)
	trees := ecs.NewFilter1[comp.Position](world).Query()
	var spared ecs.Entity
	sparedSet := false
	var toPreInfect []ecs.Entity
	for trees.Next() {
		e := trees.Entity()
		if !sparedSet {
			spared = e
			sparedSet = true
			continue
		}
		toPreInfect = append(toPreInfect, e)
	}
	trees.Close()
	for _, e := range toPreInfect {
		infectMapper.Add(e, &comp.Infected{InfectionTick: -1})
	}

	s := RandomRelease{TickOfInfection: 0, NumTrees: 5}
	s.Initialize(world)

	ecs.AddResource(world, &res.Time{Tick: 0})
	s.Update(world)

	// Only the one remaining eligible tree can have been (re)infected.
	infectedFilter := ecs.NewFilter2[comp.Position, comp.Infected](world)
	q := infectedFilter.Query()
	newlyInfected := 0
	for q.Next() {
		_, inf := q.Get()
		if inf.InfectionTick == 0 {
			newlyInfected++
			assert.Equal(t, spared, q.Entity())
		}
	}
	q.Close()
	assert.Equal(t, 1, newlyInfected)
}

func TestRandomReleaseCapsAtAvailableTrees(t *testing.T) {
	world := setupRandomReleaseWorld(t)

	// The world holds 20*20 = 400 trees; request far more than that.
	s := RandomRelease{TickOfInfection: 0, NumTrees: 1000}
	s.Initialize(world)

	ecs.AddResource(world, &res.Time{Tick: 0})
	assert.NotPanics(t, func() { s.Update(world) })

	q := ecs.NewFilter1[comp.Infected](world).Query()
	assert.Equal(t, 400, q.Count())
	q.Close()
}
