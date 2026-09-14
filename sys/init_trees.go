package sys

import (
	"math"
	"math/rand/v2"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwm-model/pwn/comp"
	"github.com/pwm-model/pwn/res"
)

// InitTrees system
type InitTrees struct {
	TreeProbability float64
}

// Initialize the system.
func (s *InitTrees) Initialize(world *ecs.World) {
	grid := ecs.GetResource[res.TreeGrid](world)
	rand := rand.New(ecs.GetResource[resource.Rand](world))

	cells := make([]comp.Position, 0, int(math.Ceil(float64(grid.Width()*grid.Height())*s.TreeProbability*1.1)))

	for x := range grid.Width() {
		for y := range grid.Height() {
			if s.TreeProbability >= 1 || rand.Float64() < s.TreeProbability {
				cells = append(cells, comp.Position{X: x, Y: y})
			}
		}
	}

	builder := ecs.NewMap1[comp.Position](world)

	cnt := 0
	builder.NewBatchFn(len(cells), func(e ecs.Entity, p *comp.Position) {
		*p = cells[cnt]
		cnt++
	})
}

// Update the system.
func (s *InitTrees) Update(w *ecs.World) {}

// Finalize the system.
func (s *InitTrees) Finalize(w *ecs.World) {}
