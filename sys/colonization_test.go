package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func setupColonizationWorld(t *testing.T, width, height, resolution int) *ecs.World {
	t.Helper()

	a := app.New()
	ecs.AddResource(a.World, &res.WorldSize{Width: width, Height: height, Resolution: resolution})

	gs := InitGrids{}
	gs.Initialize(a.World)

	return a.World
}

func TestBuildKernelSelfOnlyAtRadiusZero(t *testing.T) {
	offsets := buildKernel(0, 1)

	assert.Equal(t, []kernelOffset{{dx: 0, dy: 0, weight: 1}}, offsets)
}

func TestBuildKernelNormalizesWeights(t *testing.T) {
	// Radius 1: the center cell plus its 4 orthogonal neighbors; the
	// diagonal neighbors are farther than the radius and excluded.
	offsets := buildKernel(1, 1)
	assert.Len(t, offsets, 5)

	sum := 0.0
	var center float64
	for _, o := range offsets {
		sum += o.weight
		if o.dx == 0 && o.dy == 0 {
			center = o.weight
		}
	}
	assert.InDelta(t, 1.0, sum, 1e-9, "normalized weights must sum to 1")

	for _, o := range offsets {
		if o.dx != 0 || o.dy != 0 {
			assert.Less(t, o.weight, center, "the center cell must have the highest weight")
		}
	}
}

func TestColonizationCalcArrivalsConservesBeetleCount(t *testing.T) {
	world := setupColonizationWorld(t, 3, 3, 1)

	s := Colonization{KernelRadius: 1, KernelScale: 1, BeetlesPerTree: 10}
	s.Initialize(world)

	// Place the source away from any edge so the kernel's full support
	// stays in-bounds and beetle count is exactly conserved.
	s.density.Set(1, 1, 2)

	s.calcArrivals()

	total := 0.0
	for x := 0; x < s.arrivals.Width(); x++ {
		for y := 0; y < s.arrivals.Height(); y++ {
			total += s.arrivals.Get(x, y)
		}
	}
	assert.InDelta(t, 20.0, total, 1e-9, "beetle count (density * BeetlesPerTree) must be conserved")

	center := s.arrivals.Get(1, 1)
	neighbor := s.arrivals.Get(0, 1)
	corner := s.arrivals.Get(0, 0)
	assert.Greater(t, center, neighbor, "the source cell itself should receive the largest share")
	assert.Greater(t, neighbor, 0.0)
	assert.Equal(t, 0.0, corner, "diagonal offsets are outside a radius-1 kernel")
}

func TestColonizationCalcProbabilityOccupancyFormula(t *testing.T) {
	world := setupColonizationWorld(t, 3, 1, 1)

	s := Colonization{TreesPerBeetle: 1}
	s.Initialize(world)

	// Cell (0,0): no susceptible trees -- must be left untouched.
	s.susceptible.Set(0, 0, 0)
	s.arrivals.Set(0, 0, 5)

	// Cell (1,0): a single susceptible tree and one attempt -- that
	// attempt necessarily lands on it, so probability must be exactly 1.
	s.susceptible.Set(1, 0, 1)
	s.arrivals.Set(1, 0, 1)

	// Cell (2,0): two susceptible trees and one attempt -- the classic
	// occupancy result 1 - (1 - 1/2)^1 = 0.5.
	s.susceptible.Set(2, 0, 2)
	s.arrivals.Set(2, 0, 1)

	s.calcProbability()

	assert.Equal(t, 0.0, s.probability.Get(0, 0), "a cell with no susceptible trees must not get a probability")
	assert.InDelta(t, 1.0, s.probability.Get(1, 0), 1e-9)
	assert.InDelta(t, 0.5, s.probability.Get(2, 0), 1e-9)
}

func TestColonizationSkipsWrongTickOfYear(t *testing.T) {
	world := setupColonizationWorld(t, 10, 10, 10)
	ecs.AddResource(world, &res.Time{TickOfYear: 0})

	s := Colonization{
		TickOfYear:     5,
		KernelRadius:   0,
		KernelScale:    1,
		BeetlesPerTree: 100,
		TreesPerBeetle: 100,
	}
	s.Initialize(world)

	colonizedMap := ecs.NewMap2[comp.Position, comp.Colonized](world)
	source := colonizedMap.NewEntity(&comp.Position{X: 0, Y: 0}, &comp.Colonized{})

	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](world)
	target := damagedMap.NewEntity(&comp.Position{X: 1, Y: 1}, &comp.Damaged{})

	s.Update(world)

	colonized := ecs.NewMap1[comp.Colonized](world)
	assert.True(t, colonized.HasAll(source), "wrong tick must not touch the existing colonized tree")
	assert.False(t, colonized.HasAll(target), "wrong tick must not colonize any tree")
}

func TestColonizationColonizesSusceptibleTreeWhenBeetlesArrive(t *testing.T) {
	world := setupColonizationWorld(t, 10, 10, 10)
	ecs.AddResource(world, &res.Time{TickOfYear: 3})

	// KernelRadius 0 keeps all beetles in the source's own cell, and the
	// large Beetles/TreesPerBeetle values push the occupancy probability
	// for the cell's single susceptible tree to (effectively) exactly 1,
	// regardless of the RNG draw.
	s := Colonization{
		TickOfYear:     3,
		KernelRadius:   0,
		KernelScale:    1,
		BeetlesPerTree: 5,
		TreesPerBeetle: 5,
	}
	s.Initialize(world)

	colonizedMap := ecs.NewMap2[comp.Position, comp.Colonized](world)
	source := colonizedMap.NewEntity(&comp.Position{X: 0, Y: 0}, &comp.Colonized{})

	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](world)
	target := damagedMap.NewEntity(&comp.Position{X: 5, Y: 5}, &comp.Damaged{})

	s.Update(world)

	colonized := ecs.NewMap1[comp.Colonized](world)
	assert.True(t, colonized.HasAll(target), "the sole susceptible tree must be colonized once beetles arrive")

	// The beetles that emerged from the source tree have flown out to lay
	// their eggs elsewhere, so the source tree empties out and becomes
	// available for colonization again (coloMapper.RemoveBatch).
	assert.False(t, colonized.HasAll(source), "the source tree empties out once its beetles have dispersed")
}

func TestColonizationNoColonizationWithoutSource(t *testing.T) {
	world := setupColonizationWorld(t, 10, 10, 10)
	ecs.AddResource(world, &res.Time{TickOfYear: 3})

	s := Colonization{
		TickOfYear:     3,
		KernelRadius:   2,
		KernelScale:    1,
		BeetlesPerTree: 1000,
		TreesPerBeetle: 1000,
	}
	s.Initialize(world)

	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](world)
	targets := make([]ecs.Entity, 5)
	for i := range targets {
		targets[i] = damagedMap.NewEntity(&comp.Position{X: i, Y: 0}, &comp.Damaged{})
	}

	s.Update(world)

	colonized := ecs.NewMap1[comp.Colonized](world)
	for _, e := range targets {
		assert.False(t, colonized.HasAll(e), "without any colonized source, no beetles arrive so nothing can be colonized")
	}
}
