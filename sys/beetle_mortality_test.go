package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func TestBeetleMortalityRemovesAtOrPastTickOfDeath(t *testing.T) {
	a := app.New()
	time := res.Time{Tick: 10}
	ecs.AddResource(a.World, &time)

	s := BeetleMortality{}
	s.Initialize(a.World)

	lifeMap := ecs.NewMap1[comp.LifeExpectancy](a.World)
	// Death was in the past.
	past := lifeMap.NewEntity(&comp.LifeExpectancy{TickOfDeath: 5})
	// Death is exactly this tick (inclusive boundary).
	atThreshold := lifeMap.NewEntity(&comp.LifeExpectancy{TickOfDeath: 10})
	// Death is still in the future.
	future := lifeMap.NewEntity(&comp.LifeExpectancy{TickOfDeath: 11})

	s.Update(a.World)

	assert.False(t, a.World.Alive(past))
	assert.False(t, a.World.Alive(atThreshold))
	assert.True(t, a.World.Alive(future))
}

func TestBeetleMortalityNoDeathsNoOp(t *testing.T) {
	a := app.New()
	ecs.AddResource(a.World, &res.Time{Tick: 0})

	s := BeetleMortality{}
	s.Initialize(a.World)

	lifeMap := ecs.NewMap1[comp.LifeExpectancy](a.World)
	alive := lifeMap.NewEntity(&comp.LifeExpectancy{TickOfDeath: 100})

	assert.NotPanics(t, func() { s.Update(a.World) })
	assert.True(t, a.World.Alive(alive))
}

func TestBeetleMortalityNoEntitiesNoOp(t *testing.T) {
	a := app.New()
	ecs.AddResource(a.World, &res.Time{Tick: 0})

	s := BeetleMortality{}
	s.Initialize(a.World)

	assert.NotPanics(t, func() { s.Update(a.World) })
}

func TestBeetleMortalityToRemoveBufferResetsAcrossTicks(t *testing.T) {
	a := app.New()
	time := res.Time{Tick: 0}
	ecs.AddResource(a.World, &time)

	s := BeetleMortality{}
	s.Initialize(a.World)

	lifeMap := ecs.NewMap1[comp.LifeExpectancy](a.World)
	dead := lifeMap.NewEntity(&comp.LifeExpectancy{TickOfDeath: 0})

	s.Update(a.World)
	assert.False(t, a.World.Alive(dead))

	// If the reused toRemove slice weren't cleared, this second tick (with
	// no newly-dead entities) would try to remove the already-dead entity
	// again and panic.
	assert.NotPanics(t, func() { s.Update(a.World) })
}
