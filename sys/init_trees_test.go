package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func TestInitTrees(t *testing.T) {
	app := app.New()

	ws := res.WorldSize{Width: 100, Height: 50, Resolution: 10}
	ecs.AddResource(app.World, &ws)

	gs := InitGrids{}
	gs.Initialize(app.World)

	s := InitTrees{TreeProbability: 0.9}
	s.Initialize(app.World)

	grid := ecs.GetResource[res.TreeGrid](app.World)
	space := ecs.GetResource[res.SpaceGrid](app.World)

	q := ecs.NewFilter1[comp.Position](app.World).Query()

	// get one entity
	q.Next()
	pos := q.Get()
	entity := q.Entity()

	count := q.Count()
	q.Close()

	assert.Greater(t, count, 4400)
	assert.Less(t, count, 4600)

	assert.False(t, grid.Get(pos.X, pos.Y).IsZero())

	inCell := ecs.NewMap1[comp.InCell](app.World)
	cell := inCell.GetRelation(entity, 0)
	assert.Equal(t, space.Get(pos.X/ws.Resolution, pos.Y/ws.Resolution), cell)

	damagedQuery := ecs.NewFilter1[comp.Damaged](app.World).Query()
	assert.Equal(t, 0, damagedQuery.Count())
	damagedQuery.Close()
}

func TestInitTreesDamaged(t *testing.T) {
	app := app.New()

	ws := res.WorldSize{Width: 100, Height: 50, Resolution: 10}
	ecs.AddResource(app.World, &ws)

	gs := InitGrids{}
	gs.Initialize(app.World)

	s := InitTrees{TreeProbability: 1.0, DamagePrevalence: 0.2}
	s.Initialize(app.World)

	grid := ecs.GetResource[res.TreeGrid](app.World)
	space := ecs.GetResource[res.SpaceGrid](app.World)
	inCell := ecs.NewMap1[comp.InCell](app.World)

	// TreeProbability of 1.0 places exactly one tree per fine cell.
	totalQuery := ecs.NewFilter1[comp.Position](app.World).Query()
	total := totalQuery.Count()
	totalQuery.Close()
	assert.Equal(t, ws.Width*ws.Height, total)

	damagedQuery := ecs.NewFilter2[comp.Position, comp.Damaged](app.World).Query()
	damagedCount := 0
	for damagedQuery.Next() {
		pos, _ := damagedQuery.Get()
		entity := damagedQuery.Entity()

		// The entity grid must point to the entity actually holding this position,
		// not to a stale duplicate from another coarse cell.
		assert.Equal(t, entity, grid.Get(pos.X, pos.Y))

		// The InCell relation must match the coarse cell the position actually falls into.
		cell := inCell.GetRelation(entity, 0)
		assert.Equal(t, space.Get(pos.X/ws.Resolution, pos.Y/ws.Resolution), cell)

		damagedCount++
	}
	damagedQuery.Close()

	expected := float64(total) * s.DamagePrevalence
	assert.InDelta(t, expected, float64(damagedCount), expected*0.25)
}

func TestInitTreesColonized(t *testing.T) {
	app := app.New()

	ws := res.WorldSize{Width: 100, Height: 50, Resolution: 10}
	ecs.AddResource(app.World, &ws)

	gs := InitGrids{}
	gs.Initialize(app.World)

	s := InitTrees{TreeProbability: 1.0, DamagePrevalence: 0.2, BeetlePrevalence: 0.3}
	s.Initialize(app.World)

	grid := ecs.GetResource[res.TreeGrid](app.World)

	totalQuery := ecs.NewFilter1[comp.Position](app.World).Query()
	total := totalQuery.Count()
	totalQuery.Close()

	// Every colonized tree must also be marked damaged: the count of trees
	// with both components must match the count of trees with Colonized alone.
	allColonizedQuery := ecs.NewFilter2[comp.Position, comp.Colonized](app.World).Query()
	allColonizedCount := allColonizedQuery.Count()
	allColonizedQuery.Close()

	colonizedQuery := ecs.NewFilter3[comp.Position, comp.Damaged, comp.Colonized](app.World).Query()
	colonizedCount := 0
	for colonizedQuery.Next() {
		pos, _, _ := colonizedQuery.Get()
		entity := colonizedQuery.Entity()

		// The entity grid must point to the entity actually holding this position,
		// not to a stale duplicate from another coarse cell.
		assert.Equal(t, entity, grid.Get(pos.X, pos.Y))

		colonizedCount++
	}
	colonizedQuery.Close()

	assert.Equal(t, allColonizedCount, colonizedCount)

	expectedColonized := float64(total) * s.DamagePrevalence * s.BeetlePrevalence
	assert.InDelta(t, expectedColonized, float64(colonizedCount), expectedColonized*0.25)

	damagedOnlyQuery := ecs.NewFilter2[comp.Position, comp.Damaged](app.World).Without(ecs.C[comp.Colonized]()).Query()
	damagedOnlyCount := damagedOnlyQuery.Count()
	damagedOnlyQuery.Close()

	expectedDamagedOnly := float64(total) * s.DamagePrevalence * (1 - s.BeetlePrevalence)
	assert.InDelta(t, expectedDamagedOnly, float64(damagedOnlyCount), expectedDamagedOnly*0.25)
}
