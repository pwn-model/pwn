package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func TestBeetleEmergenceSkipsWrongTickOfYear(t *testing.T) {
	a := app.New()
	ecs.AddResource(a.World, &res.Time{TickOfYear: 1})

	s := BeetleEmergence{TickOfYear: 0, BeetlesPerTree: 5, LifeExpectancy: 10}
	s.Initialize(a.World)

	sourceMap := ecs.NewMap3[comp.Position, comp.Damaged, comp.Infected](a.World)
	sourceMap.NewEntity(&comp.Position{X: 1, Y: 2}, &comp.Damaged{}, &comp.Infected{})

	s.Update(a.World)

	q := ecs.NewFilter1[comp.BeetlePosition](a.World).Query()
	assert.Equal(t, 0, q.Count())
	q.Close()
}

func TestBeetleEmergenceCreatesBeetlesFromSourceTrees(t *testing.T) {
	a := app.New()
	time := res.Time{TickOfYear: 3, Tick: 20}
	ecs.AddResource(a.World, &time)

	s := BeetleEmergence{TickOfYear: 3, BeetlesPerTree: 4, LifeExpectancy: 10}
	s.Initialize(a.World)

	sourceMap := ecs.NewMap3[comp.Position, comp.Damaged, comp.Infected](a.World)
	sourceMap.NewEntity(&comp.Position{X: 1, Y: 2}, &comp.Damaged{}, &comp.Infected{})
	sourceMap.NewEntity(&comp.Position{X: 5, Y: 6}, &comp.Damaged{}, &comp.Infected{})

	s.Update(a.World)

	beetleFilter := ecs.NewFilter2[comp.BeetlePosition, comp.LifeExpectancy](a.World)
	q := beetleFilter.Query()
	assert.Equal(t, 8, q.Count())

	counts := map[[2]int]int{}
	for q.Next() {
		bp, le := q.Get()
		counts[[2]int{bp.X, bp.Y}]++
		// ExpFloat64 is never negative, so death can never precede the
		// current tick.
		assert.GreaterOrEqual(t, le.TickOfDeath, time.Tick)
	}
	q.Close()

	assert.Equal(t, 4, counts[[2]int{1, 2}], "each source tree must emerge exactly BeetlesPerTree beetles")
	assert.Equal(t, 4, counts[[2]int{5, 6}], "each source tree must emerge exactly BeetlesPerTree beetles")
}

func TestBeetleEmergenceRequiresBothDamagedAndInfected(t *testing.T) {
	a := app.New()
	ecs.AddResource(a.World, &res.Time{TickOfYear: 3})

	s := BeetleEmergence{TickOfYear: 3, BeetlesPerTree: 5, LifeExpectancy: 10}
	s.Initialize(a.World)

	damagedOnly := ecs.NewMap2[comp.Position, comp.Damaged](a.World)
	damagedOnly.NewEntity(&comp.Position{X: 1, Y: 1}, &comp.Damaged{})

	infectedOnly := ecs.NewMap2[comp.Position, comp.Infected](a.World)
	infectedOnly.NewEntity(&comp.Position{X: 2, Y: 2}, &comp.Infected{})

	s.Update(a.World)

	q := ecs.NewFilter1[comp.BeetlePosition](a.World).Query()
	assert.Equal(t, 0, q.Count(), "only trees with both Damaged and Infected are emergence sources")
	q.Close()
}

func TestBeetleEmergenceNoSourceTreesNoOp(t *testing.T) {
	a := app.New()
	ecs.AddResource(a.World, &res.Time{TickOfYear: 3})

	s := BeetleEmergence{TickOfYear: 3, BeetlesPerTree: 5, LifeExpectancy: 10}
	s.Initialize(a.World)

	assert.NotPanics(t, func() { s.Update(a.World) })

	q := ecs.NewFilter1[comp.BeetlePosition](a.World).Query()
	assert.Equal(t, 0, q.Count())
	q.Close()
}

func TestBeetleEmergenceSourceBufferDoesNotLeakAcrossTicks(t *testing.T) {
	a := app.New()
	time := res.Time{TickOfYear: 3}
	ecs.AddResource(a.World, &time)

	s := BeetleEmergence{TickOfYear: 3, BeetlesPerTree: 2, LifeExpectancy: 10}
	s.Initialize(a.World)

	sourceMap := ecs.NewMap3[comp.Position, comp.Damaged, comp.Infected](a.World)
	sourceMap.NewEntity(&comp.Position{X: 1, Y: 1}, &comp.Damaged{}, &comp.Infected{})

	s.Update(a.World)
	s.Update(a.World)

	// The reused sourceTrees slice must be cleared after each Update; if it
	// weren't, the second tick's query would append onto the first tick's
	// leftover entries and over-count emerged beetles.
	q := ecs.NewFilter1[comp.BeetlePosition](a.World).Query()
	assert.Equal(t, 4, q.Count())
	q.Close()
}
