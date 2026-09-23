package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func setupRandomInfectionWorld(t *testing.T) *ecs.World {
	t.Helper()

	a := app.New()

	// 2x2 coarse grid, 100 trees per coarse cell.
	ws := res.NewWorldSize(200, 200, 10, 100)
	ecs.AddResource(a.World, &ws)

	gs := InitGrids{}
	gs.Initialize(a.World)

	ts := InitTrees{CellProbability: 1.0, TreeProbability: 1.0}
	ts.Initialize(a.World)

	return a.World
}

func TestRandomInfection(t *testing.T) {
	world := setupRandomInfectionWorld(t)

	s := RandomInfection{TickOfInfection: 3, NumTrees: 10, CellX: 0, CellY: 0}
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

	ws := ecs.GetResource[res.WorldSize](world)
	space := ecs.GetResource[res.SpaceGrid](world)
	inCell := ecs.NewMap1[comp.InCell](world)
	target := space.Get(0, 0)

	posQuery := ecs.NewFilter2[comp.Position, comp.Infected](world).Query()
	count := 0
	for posQuery.Next() {
		pos, inf := posQuery.Get()
		entity := posQuery.Entity()

		assert.Equal(t, 3, inf.InfectionTick)
		assert.Equal(t, target, inCell.GetRelation(entity, 0))
		assert.Less(t, pos.X, ws.Resolution())
		assert.Less(t, pos.Y, ws.Resolution())
		count++
	}
	posQuery.Close()
	assert.Equal(t, 10, count)

	// Later ticks must not infect further trees.
	time.Tick = 4
	s.Update(world)

	q = infected.Query()
	assert.Equal(t, 10, q.Count())
	q.Close()
}

func TestRandomInfectionSkipsIneligibleTrees(t *testing.T) {
	world := setupRandomInfectionWorld(t)

	space := ecs.GetResource[res.SpaceGrid](world)
	target := space.Get(0, 0)

	// Mark all trees in the target cell as already infected, except one.
	infectMapper := ecs.NewMap1[comp.Infected](world)
	posInCell := ecs.NewFilter2[comp.Position, comp.InCell](world).Query(ecs.RelIdx(1, target))
	var spared ecs.Entity
	sparedSet := false
	var toPreInfect []ecs.Entity
	for posInCell.Next() {
		e := posInCell.Entity()
		if !sparedSet {
			spared = e
			sparedSet = true
			continue
		}
		toPreInfect = append(toPreInfect, e)
	}
	posInCell.Close()
	for _, e := range toPreInfect {
		infectMapper.Add(e, &comp.Infected{InfectionTick: -1})
	}

	s := RandomInfection{TickOfInfection: 0, NumTrees: 5, CellX: 0, CellY: 0}
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

func TestRandomInfectionCapsAtAvailableTrees(t *testing.T) {
	world := setupRandomInfectionWorld(t)

	// Coarse cell (0, 0) holds 10*10 = 100 trees; request far more than that.
	s := RandomInfection{TickOfInfection: 0, NumTrees: 1000, CellX: 0, CellY: 0}
	s.Initialize(world)

	ecs.AddResource(world, &res.Time{Tick: 0})
	assert.NotPanics(t, func() { s.Update(world) })

	q := ecs.NewFilter1[comp.Infected](world).Query()
	assert.Equal(t, 100, q.Count())
	q.Close()
}

func TestRandomInfectionOnlyTargetsSpecifiedCell(t *testing.T) {
	world := setupRandomInfectionWorld(t)

	s := RandomInfection{TickOfInfection: 0, NumTrees: 10, CellX: 1, CellY: 1}
	s.Initialize(world)

	ecs.AddResource(world, &res.Time{Tick: 0})
	s.Update(world)

	ws := ecs.GetResource[res.WorldSize](world)
	q := ecs.NewFilter2[comp.Position, comp.Infected](world).Query()
	for q.Next() {
		pos, _ := q.Get()
		assert.GreaterOrEqual(t, pos.X, ws.Resolution())
		assert.GreaterOrEqual(t, pos.Y, ws.Resolution())
	}
	q.Close()
}
