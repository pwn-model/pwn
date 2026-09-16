package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/stretchr/testify/assert"
)

func TestDiseaseCourse(t *testing.T) {
	a := app.New()

	s := DiseaseCourse{TicksToDamage: 5}
	s.Initialize(a.World)

	infected := ecs.NewMap1[comp.NematodeInfected](a.World)
	damaged := ecs.NewMap1[comp.Damaged](a.World)

	// Elapsed time (tick - InfectionTick) is within TicksToDamage.
	inWindow := infected.NewEntity(&comp.NematodeInfected{InfectionTick: 8})
	// Elapsed time exceeds TicksToDamage.
	outOfWindow := infected.NewEntity(&comp.NematodeInfected{InfectionTick: 0})

	tick := ecs.GetResource[resource.Tick](a.World)
	tick.Tick = 10

	s.Update(a.World)

	assert.True(t, damaged.HasAll(inWindow))
	assert.False(t, damaged.HasAll(outOfWindow))
}

func TestDiseaseCourseSkipsAlreadyDamaged(t *testing.T) {
	a := app.New()

	s := DiseaseCourse{TicksToDamage: 5}
	s.Initialize(a.World)

	infected := ecs.NewMap2[comp.NematodeInfected, comp.Damaged](a.World)
	entity := infected.NewEntity(&comp.NematodeInfected{InfectionTick: 8}, &comp.Damaged{})

	tick := ecs.GetResource[resource.Tick](a.World)
	tick.Tick = 10

	// Must not panic: entities already carrying comp.Damaged are excluded
	// by the system's filter, so Update must not try to add it again.
	assert.NotPanics(t, func() { s.Update(a.World) })

	damaged := ecs.NewMap1[comp.Damaged](a.World)
	assert.True(t, damaged.HasAll(entity))
}

func TestDiseaseCourseIdempotentAcrossTicks(t *testing.T) {
	a := app.New()

	s := DiseaseCourse{TicksToDamage: 5}
	s.Initialize(a.World)

	infected := ecs.NewMap1[comp.NematodeInfected](a.World)
	entity := infected.NewEntity(&comp.NematodeInfected{InfectionTick: 0})

	tick := ecs.GetResource[resource.Tick](a.World)

	tick.Tick = 0
	s.Update(a.World)

	damaged := ecs.NewMap1[comp.Damaged](a.World)
	assert.True(t, damaged.HasAll(entity))

	// A later Update must not try to add comp.Damaged again to an
	// already-damaged entity.
	tick.Tick = 1
	assert.NotPanics(t, func() { s.Update(a.World) })
}
