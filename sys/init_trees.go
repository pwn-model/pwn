package sys

import (
	"math/rand/v2"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
)

// InitTrees system
type InitTrees struct {
	TreeProbability float64
}

// Initialize the system.
func (s *InitTrees) Initialize(world *ecs.World) {
	ws := ecs.GetResource[res.WorldSize](world)
	grid := ecs.GetResource[res.SpaceGrid](world)
	trees := ecs.GetResource[res.EntityGrid](world)
	rand := rand.New(ecs.GetResource[resource.Rand](world))

	builder := ecs.NewMap2[comp.Position, comp.InCell](world)
	cells := make([]comp.Position, 0, ws.Resolution*ws.Resolution)

	for x := range grid.Width() {
		for y := range grid.Height() {
			cells := cells[:0]
			cell := grid.Get(x, y)

			for dx := range ws.Resolution {
				for dy := range ws.Resolution {
					if s.TreeProbability < 1.0 && rand.Float64() > s.TreeProbability {
						continue
					}
					xx := x*ws.Resolution + dx
					yy := y*ws.Resolution + dy
					cells = append(cells, comp.Position{X: xx, Y: yy})
				}
			}

			cnt := 0
			builder.NewBatchFn(len(cells), func(e ecs.Entity, p *comp.Position, _ *comp.InCell) {
				*p = cells[cnt]
				trees.Set(p.X, p.Y, e)
				cnt++
			}, ecs.RelIdx(1, cell))
		}
	}
}

// Update the system.
func (s *InitTrees) Update(_ *ecs.World) {}

// Finalize the system.
func (s *InitTrees) Finalize(_ *ecs.World) {}
