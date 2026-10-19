package sys

import (
	"math"
	"math/rand/v2"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/util"
)

func init() {
	config.Register[InitTrees]()
}

// InitTrees system
type InitTrees struct {
	Km2Probability   float64 `yaml:"km2_probability"`
	TreeProbability  float64 `yaml:"tree_probability"`
	DamagePrevalence float64 `yaml:"damage_prevalence"`
	BeetlePrevalence float64 `yaml:"beetle_prevalence"`
}

// Initialize the system.
func (s *InitTrees) Initialize(world *ecs.World) {
	trees := ecs.GetResource[res.TreeGrid](world)
	rand := rand.New(ecs.GetResource[resource.Rand](world))

	builderDefault := ecs.NewMap1[comp.Position](world)
	builderDamaged := ecs.NewMap2[comp.Position, comp.Damaged](world)
	builderColonized := ecs.NewMap3[comp.Position, comp.Damaged, comp.Colonized](world)

	cellsPerKm2 := int(math.Round(1000.0 / float64(trees.CellSize())))
	cellsSize := trees.CellSize() * cellsPerKm2

	cells := make([]comp.Position, 0, trees.Width()*trees.Height())
	damaged := make([]comp.Position, 0, int(math.Ceil(float64(cap(cells))*s.DamagePrevalence*1.2)))
	colonized := make([]comp.Position, 0, int(math.Ceil(float64(cap(cells))*s.DamagePrevalence*s.BeetlePrevalence*1.2)))

	hectares := res.NewGrid[bool](util.CeilDiv(trees.Width(), cellsPerKm2), util.CeilDiv(trees.Height(), cellsPerKm2), cellsSize)
	for x := range hectares.Width() {
		for y := range hectares.Height() {
			if rand.Float64() < s.Km2Probability {
				hectares.Set(x, y, true)
			}
		}
	}

	for x := range trees.Width() {
		for y := range trees.Height() {
			if !hectares.Get(x/cellsPerKm2, y/cellsPerKm2) {
				continue
			}
			if s.TreeProbability < 1.0 && rand.Float64() > s.TreeProbability {
				continue
			}
			r := rand.Float64()
			if r < s.DamagePrevalence {
				if r < s.DamagePrevalence*s.BeetlePrevalence {
					colonized = append(colonized, comp.Position{X: x, Y: y})
				} else {
					damaged = append(damaged, comp.Position{X: x, Y: y})
				}
			} else {
				cells = append(cells, comp.Position{X: x, Y: y})
			}
		}
	}

	cnt := 0
	builderDefault.NewBatchFn(len(cells), func(e ecs.Entity, p *comp.Position) {
		*p = cells[cnt]
		trees.Set(p.X, p.Y, e)
		cnt++
	})

	cnt = 0
	builderDamaged.NewBatchFn(len(damaged), func(e ecs.Entity, p *comp.Position, _ *comp.Damaged) {
		*p = damaged[cnt]
		trees.Set(p.X, p.Y, e)
		cnt++
	})

	cnt = 0
	builderColonized.NewBatchFn(len(colonized), func(e ecs.Entity, p *comp.Position, _ *comp.Damaged, _ *comp.Colonized) {
		*p = colonized[cnt]
		trees.Set(p.X, p.Y, e)
		cnt++
	})
}

// Update the system.
func (s *InitTrees) Update(_ *ecs.World) {}

// Finalize the system.
func (s *InitTrees) Finalize(_ *ecs.World) {}
