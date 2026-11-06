package sys

import (
	"math"
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/util"
	"github.com/stretchr/testify/assert"
)

// The published sources are exactly those util.PropagateOctants computes
// from seeds counted by brute force, for random trees incl. near the edges.
func TestTreeAttractionOctantsMatchesBruteForceSeeds(t *testing.T) {
	a := app.New()
	ws := res.NewWorldSize(300, 200, 10)
	ecs.AddResource(a.World, &ws)
	ecs.AddResource(a.World, &res.Time{TickOfYear: 0})
	posMap := ecs.NewMap1[comp.Position](a.World)

	src := res.NewXoshiro256pp(1)
	w, h := ws.Width(), ws.Height()
	for range 200 {
		posMap.NewEntity(&comp.Position{X: int(src.Uint64() % uint64(w)), Y: int(src.Uint64() % uint64(h))})
	}

	s := TreeAttractionOctants{TickOfYear: 0, HalfDistance: 50, DensityRadius: 40, DensityWeight: 1.5}
	s.Initialize(a.World)
	s.Update(a.World)

	octants := ecs.GetResource[res.HealthyTreeAttractionOctants](a.World)

	r := 4
	maxCount := float64((2*r + 1) * (2*r + 1))
	seeds := res.NewGrid[float64](w, h, ws.CellSize())
	trees := 0
	for x := range w {
		for y := range h {
			if !octants.Presence.Get(x, y) {
				continue
			}
			trees++
			count := 0
			for xx := max(x-r, 0); xx <= min(x+r, w-1); xx++ {
				for yy := max(y-r, 0); yy <= min(y+r, h-1); yy++ {
					if octants.Presence.Get(xx, yy) {
						count++
					}
				}
			}
			seeds.Set(x, y, math.Pow(float64(count)/maxCount, 1.5))
		}
	}
	expected := res.NewGrid[res.OctantSources](w, h, ws.CellSize())
	buf := util.NewOctantBuffers(h)
	util.PropagateOctants(&expected, &seeds, 50, &buf)

	assert.Greater(t, trees, 0)
	mismatches := 0
	for x := range w {
		for y := range h {
			if octants.Sources.Get(x, y) != expected.Get(x, y) {
				mismatches++
			}
		}
	}
	assert.Equal(t, 0, mismatches)
}

// Two instances, one for healthy and one for damaged trees, publish
// separate resources, each driven only by its own tree type.
func TestTreeAttractionOctantsHealthyAndDamagedInstancesAreIndependent(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	posMap.NewEntity(&comp.Position{X: 50, Y: 25})
	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](world)
	damagedMap.NewEntity(&comp.Position{X: 80, Y: 25}, &comp.Damaged{})

	healthy := TreeAttractionOctants{TickOfYear: 0, HalfDistance: 50, DensityRadius: 20, DensityWeight: 1}
	damaged := TreeAttractionOctants{TickOfYear: 0, DamagedTrees: true, HalfDistance: 50, DensityRadius: 20, DensityWeight: 1}
	healthy.Initialize(world)
	damaged.Initialize(world)
	healthy.Update(world)
	damaged.Update(world)

	h := ecs.GetResource[res.HealthyTreeAttractionOctants](world)
	d := ecs.GetResource[res.DamagedTreeAttractionOctants](world)
	assert.True(t, h.Presence.Get(50, 25))
	assert.False(t, h.Presence.Get(80, 25))
	assert.True(t, d.Presence.Get(80, 25))
	assert.False(t, d.Presence.Get(50, 25))

	// From (60, 25), the healthy tree is 10 cells towards -x (octant 4), the
	// damaged one 20 cells towards +x (octant 0).
	hr, dr := h.Sources.Get(60, 25), d.Sources.Get(60, 25)
	assert.Equal(t, [2]int16{-10, 0}, hr.Offsets[4])
	assert.Equal(t, [2]int16{20, 0}, dr.Offsets[0])
	for k := range 8 {
		if k != 4 {
			assert.Equal(t, float32(0), hr.Attraction[k])
		}
		if k != 0 {
			assert.Equal(t, float32(0), dr.Attraction[k])
		}
	}

	// A tree's own cell is not its own source.
	assert.Equal(t, res.OctantSources{}, h.Sources.Get(50, 25))
}

// Sources and presence are only updated on TickOfYear.
func TestTreeAttractionOctantsUpdatesOnlyOnTickOfYear(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	e := posMap.NewEntity(&comp.Position{X: 50, Y: 25})

	s := TreeAttractionOctants{TickOfYear: 5, HalfDistance: 50, DensityRadius: 20, DensityWeight: 1}
	s.Initialize(world)
	octants := ecs.GetResource[res.HealthyTreeAttractionOctants](world)

	s.Update(world)
	assert.False(t, octants.Presence.Get(50, 25))
	assert.Equal(t, res.OctantSources{}, octants.Sources.Get(60, 25))

	timeRes := ecs.GetResource[res.Time](world)
	timeRes.TickOfYear = 5
	s.Update(world)
	assert.True(t, octants.Presence.Get(50, 25))
	assert.Greater(t, octants.Sources.Get(60, 25).Attraction[4], float32(0))

	// A tree becoming damaged is not reflected outside TickOfYear...
	ecs.NewMap[comp.Damaged](world).Add(e, &comp.Damaged{})
	timeRes.TickOfYear = 6
	s.Update(world)
	assert.True(t, octants.Presence.Get(50, 25))

	// ...but on the next TickOfYear.
	timeRes.TickOfYear = 5
	s.Update(world)
	assert.False(t, octants.Presence.Get(50, 25))
	assert.Equal(t, res.OctantSources{}, octants.Sources.Get(60, 25))
}

func TestTreeAttractionOctantsZeroWeightSkipsDensityRadiusValidation(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	posMap.NewEntity(&comp.Position{X: 50, Y: 25})

	s := TreeAttractionOctants{TickOfYear: 0, HalfDistance: 50, DensityRadius: 7, DensityWeight: 0}
	s.Initialize(world)
	s.Update(world)

	// Seed 1, decayed over 10 cells (100 m) at HalfDistance 50 m.
	octants := ecs.GetResource[res.HealthyTreeAttractionOctants](world)
	assert.InDelta(t, 0.25, float64(octants.Sources.Get(60, 25).Attraction[4]), 1e-6)

	world, _ = setupTreeAttractionWorld(t)
	s = TreeAttractionOctants{TickOfYear: 0, HalfDistance: 50, DensityRadius: 7, DensityWeight: 1}
	assert.Panics(t, func() { s.Initialize(world) })
}

func TestTreeAttractionOctantsRejectsWorldsTooLargeForInt16Offsets(t *testing.T) {
	newWorld := func(width, height int) *ecs.World {
		a := app.New()
		ws := res.NewWorldSize(width, height, 1)
		ecs.AddResource(a.World, &ws)
		ecs.AddResource(a.World, &res.Time{TickOfYear: 0})
		return a.World
	}
	for _, size := range [][2]int{{32768, 1}, {1, 32768}} {
		s := TreeAttractionOctants{HalfDistance: 50}
		assert.NotPanics(t, func() { s.Initialize(newWorld(size[0], size[1])) })
	}
	for _, size := range [][2]int{{32769, 1}, {1, 32769}} {
		s := TreeAttractionOctants{HalfDistance: 50}
		assert.Panics(t, func() { s.Initialize(newWorld(size[0], size[1])) })
	}
}
