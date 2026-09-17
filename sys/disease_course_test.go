package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func TestDiseaseCourse(t *testing.T) {
	a := app.New()

	s := DiseaseCourse{TicksToDamage: 5}
	s.Initialize(a.World)

	infected := ecs.NewMap1[comp.Infected](a.World)
	damaged := ecs.NewMap1[comp.Damaged](a.World)

	// Elapsed time (tick - InfectionTick) exceeds TicksToDamage.
	longInfected := infected.NewEntity(&comp.Infected{InfectionTick: 0})
	// Elapsed time equals TicksToDamage exactly (inclusive boundary).
	atThreshold := infected.NewEntity(&comp.Infected{InfectionTick: 5})
	// Elapsed time is below TicksToDamage.
	recentlyInfected := infected.NewEntity(&comp.Infected{InfectionTick: 8})

	time := res.Time{Tick: 10}
	ecs.AddResource(a.World, &time)

	s.Update(a.World)

	assert.True(t, damaged.HasAll(longInfected))
	assert.True(t, damaged.HasAll(atThreshold))
	assert.False(t, damaged.HasAll(recentlyInfected))
}

func TestDiseaseCourseSkipsAlreadyDamaged(t *testing.T) {
	a := app.New()

	s := DiseaseCourse{TicksToDamage: 5}
	s.Initialize(a.World)

	infected := ecs.NewMap2[comp.Infected, comp.Damaged](a.World)
	// Elapsed time exceeds TicksToDamage, so this entity would be eligible
	// for damage if it weren't already damaged.
	entity := infected.NewEntity(&comp.Infected{InfectionTick: 0}, &comp.Damaged{})

	time := res.Time{Tick: 10}
	ecs.AddResource(a.World, &time)

	// Must not panic: entities already carrying comp.Damaged are excluded
	// by the system's filter, so Update must not try to add it again.
	assert.NotPanics(t, func() { s.Update(a.World) })

	damaged := ecs.NewMap1[comp.Damaged](a.World)
	assert.True(t, damaged.HasAll(entity))
}

func TestDiseaseCourseIdempotentAcrossTicks(t *testing.T) {
	a := app.New()

	s := DiseaseCourse{TicksToDamage: 0}
	s.Initialize(a.World)

	infected := ecs.NewMap1[comp.Infected](a.World)
	entity := infected.NewEntity(&comp.Infected{InfectionTick: 0})

	time := res.Time{Tick: 0}
	ecs.AddResource(a.World, &time)

	s.Update(a.World)

	damaged := ecs.NewMap1[comp.Damaged](a.World)
	assert.True(t, damaged.HasAll(entity))

	// A later Update must not try to add comp.Damaged again to an
	// already-damaged entity.
	time.Tick = 1
	assert.NotPanics(t, func() { s.Update(a.World) })
}
