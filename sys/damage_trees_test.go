package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func TestDamageTreesSkipsWrongTickOfYear(t *testing.T) {
	a := app.New()
	ecs.AddResource(a.World, &res.Time{TickOfYear: 1})

	s := DamageTrees{TickOfYear: 0, DamageProbability: 1, RemovalProbability: 1}
	s.Initialize(a.World)

	posMap := ecs.NewMap1[comp.Position](a.World)
	healthy := posMap.NewEntity(&comp.Position{})

	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](a.World)
	damaged := damagedMap.NewEntity(&comp.Position{}, &comp.Damaged{})

	s.Update(a.World)

	damagedMap2 := ecs.NewMap1[comp.Damaged](a.World)
	assert.False(t, damagedMap2.HasAll(healthy))
	assert.True(t, a.World.Alive(damaged))
}

func TestDamageTreesDamagesAllWithProbabilityOne(t *testing.T) {
	a := app.New()
	ecs.AddResource(a.World, &res.Time{TickOfYear: 3})

	s := DamageTrees{TickOfYear: 3, DamageProbability: 1, RemovalProbability: 0}
	s.Initialize(a.World)

	posMap := ecs.NewMap1[comp.Position](a.World)
	entities := make([]ecs.Entity, 5)
	for i := range entities {
		entities[i] = posMap.NewEntity(&comp.Position{X: i})
	}

	s.Update(a.World)

	damaged := ecs.NewMap1[comp.Damaged](a.World)
	for _, e := range entities {
		assert.True(t, damaged.HasAll(e))
	}
}

func TestDamageTreesZeroProbabilitiesNoOp(t *testing.T) {
	a := app.New()
	ecs.AddResource(a.World, &res.Time{TickOfYear: 3})

	s := DamageTrees{TickOfYear: 3, DamageProbability: 0, RemovalProbability: 0}
	s.Initialize(a.World)

	posMap := ecs.NewMap1[comp.Position](a.World)
	healthy := posMap.NewEntity(&comp.Position{})

	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](a.World)
	damagedEntity := damagedMap.NewEntity(&comp.Position{}, &comp.Damaged{})

	s.Update(a.World)

	damaged := ecs.NewMap1[comp.Damaged](a.World)
	assert.False(t, damaged.HasAll(healthy))
	assert.True(t, a.World.Alive(damagedEntity))
	assert.True(t, damaged.HasAll(damagedEntity))
}

func TestDamageTreesRemovesAllWithProbabilityOne(t *testing.T) {
	a := app.New()
	ecs.AddResource(a.World, &res.Time{TickOfYear: 3})

	grid := res.TreeGrid{Grid: res.NewGrid[ecs.Entity](5, 1, 10)}
	ecs.AddResource(a.World, &grid)

	s := DamageTrees{TickOfYear: 3, DamageProbability: 0, RemovalProbability: 1}
	s.Initialize(a.World)

	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](a.World)
	entities := make([]ecs.Entity, 5)
	for i := range entities {
		pos := comp.Position{X: i}
		entities[i] = damagedMap.NewEntity(&pos, &comp.Damaged{})
		grid.Set(pos.X, pos.Y, entities[i])
	}

	s.Update(a.World)

	for i, e := range entities {
		assert.False(t, a.World.Alive(e))
		assert.True(t, grid.Get(i, 0).IsZero())
	}
}

func TestDamageTreesRequiresPositionToDamage(t *testing.T) {
	a := app.New()
	ecs.AddResource(a.World, &res.Time{TickOfYear: 3})

	s := DamageTrees{TickOfYear: 3, DamageProbability: 1, RemovalProbability: 0}
	s.Initialize(a.World)

	// An entity without comp.Position is ineligible for damage, regardless
	// of DamageProbability.
	noPos := a.World.NewEntity()

	s.Update(a.World)

	damaged := ecs.NewMap1[comp.Damaged](a.World)
	assert.False(t, damaged.HasAll(noPos))
}
