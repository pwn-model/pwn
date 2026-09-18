package sys

import (
	"math"
	"math/rand/v2"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
)

// InitTrees system
type InitTrees struct {
	TreeProbability  float64
	DamagePrevalence float64
	BeetlePrevalence float64
}

// Initialize the system.
func (s *InitTrees) Initialize(world *ecs.World) {
	ws := ecs.GetResource[res.WorldSize](world)
	grid := ecs.GetResource[res.SpaceGrid](world)
	trees := ecs.GetResource[res.EntityGrid](world)
	rand := rand.New(ecs.GetResource[resource.Rand](world))

	builderDefault := ecs.NewMap2[comp.Position, comp.InCell](world)
	builderDamaged := ecs.NewMap3[comp.Position, comp.InCell, comp.Damaged](world)
	builderColonized := ecs.NewMap4[comp.Position, comp.InCell, comp.Damaged, comp.Colonized](world)

	cells := make([]comp.Position, 0, ws.Resolution*ws.Resolution)
	damaged := make([]comp.Position, 0, int(math.Ceil(float64(ws.Resolution*ws.Resolution)*s.DamagePrevalence*1.2)))
	colonized := make([]comp.Position, 0, int(math.Ceil(float64(ws.Resolution*ws.Resolution)*s.DamagePrevalence*s.BeetlePrevalence*1.2)))

	for x := range grid.Width() {
		for y := range grid.Height() {
			cells := cells[:0]
			damaged := damaged[:0]
			colonized := colonized[:0]
			cell := grid.Get(x, y)

			for dx := range ws.Resolution {
				for dy := range ws.Resolution {
					if s.TreeProbability < 1.0 && rand.Float64() > s.TreeProbability {
						continue
					}
					xx := x*ws.Resolution + dx
					yy := y*ws.Resolution + dy
					r := rand.Float64()
					if r < s.DamagePrevalence {
						if r < s.DamagePrevalence*s.BeetlePrevalence {
							colonized = append(colonized, comp.Position{X: xx, Y: yy})
						} else {
							damaged = append(damaged, comp.Position{X: xx, Y: yy})
						}
					} else {
						cells = append(cells, comp.Position{X: xx, Y: yy})
					}
				}
			}

			cnt := 0
			builderDefault.NewBatchFn(len(cells), func(e ecs.Entity, p *comp.Position, _ *comp.InCell) {
				*p = cells[cnt]
				trees.Set(p.X, p.Y, e)
				cnt++
			}, ecs.RelIdx(1, cell))

			cnt = 0
			builderDamaged.NewBatchFn(len(damaged), func(e ecs.Entity, p *comp.Position, _ *comp.InCell, _ *comp.Damaged) {
				*p = damaged[cnt]
				trees.Set(p.X, p.Y, e)
				cnt++
			}, ecs.RelIdx(1, cell))

			cnt = 0
			builderColonized.NewBatchFn(len(colonized), func(e ecs.Entity, p *comp.Position, _ *comp.InCell, _ *comp.Damaged, _ *comp.Colonized) {
				*p = colonized[cnt]
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
