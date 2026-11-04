package sys

import (
	"math"
	"math/rand/v2"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[InitTreeStands]()
}

// TreeStand is an axis-aligned rectangular forest stand for [InitTreeStands].
//
// Bounds are in meters. A grid cell belongs to the stand if its center is
// within [XMin, XMax) and [YMin, YMax). The other parameters have the same
// meaning as for [InitTrees] within a forested km² block.
type TreeStand struct {
	XMin             float64 `yaml:"x_min"`
	YMin             float64 `yaml:"y_min"`
	XMax             float64 `yaml:"x_max"`
	YMax             float64 `yaml:"y_max"`
	TreeProbability  float64 `yaml:"tree_probability"`
	DamagePrevalence float64 `yaml:"damage_prevalence"`
	BeetlePrevalence float64 `yaml:"beetle_prevalence"`
}

// InitTreeStands system initializes trees and their damage and beetle
// colonization state, like [InitTrees], but from a list of rectangular
// [TreeStand]s with their own parameters instead of randomly forested km²
// blocks.
//
// Where stands overlap, later stands overwrite earlier ones. Cells outside of
// all stands remain empty.
type InitTreeStands struct {
	Stands []TreeStand `yaml:"stands"`
}

// standCellRange returns the range [first, last) of 0-based cell indices
// along one axis whose centers are within [lo, hi). Computed like in the
// sibling Julia implementation (from 1-based indices), so that rounding is
// identical.
func standCellRange(lo, hi float64, cellSize, n int) (int, int) {
	first := int(math.Max(1, math.Min(math.Ceil(lo/float64(cellSize)+0.5), float64(n+1)))) - 1
	last := int(math.Max(1, math.Min(math.Ceil(hi/float64(cellSize)+0.5), float64(n+1)))) - 1
	return first, last
}

// Initialize the system.
func (s *InitTreeStands) Initialize(world *ecs.World) {
	trees := ecs.GetResource[res.TreeGrid](world)
	rand := rand.New(ecs.GetResource[resource.Rand](world))

	builderDefault := ecs.NewMap1[comp.Position](world)
	builderDamaged := ecs.NewMap2[comp.Position, comp.Damaged](world)
	builderColonized := ecs.NewMap3[comp.Position, comp.Damaged, comp.Colonized](world)

	width, height, cs := trees.Width(), trees.Height(), trees.CellSize()

	// Index of the stand each cell belongs to, plus one; 0 for none.
	// Later stands overwrite earlier ones.
	standIndex := res.NewGrid[int](width, height, cs)
	for i, stand := range s.Stands {
		x1, x2 := standCellRange(stand.XMin, stand.XMax, cs, width)
		y1, y2 := standCellRange(stand.YMin, stand.YMax, cs, height)
		for x := x1; x < x2; x++ {
			for y := y1; y < y2; y++ {
				standIndex.Set(x, y, i+1)
			}
		}
	}

	var cells, damaged, colonized []comp.Position

	for x := range width {
		for y := range height {
			i := standIndex.Get(x, y)
			if i == 0 {
				continue
			}
			stand := &s.Stands[i-1]
			if stand.TreeProbability < 1.0 && rand.Float64() > stand.TreeProbability {
				continue
			}
			r := rand.Float64()
			if r < stand.DamagePrevalence {
				if r < stand.DamagePrevalence*stand.BeetlePrevalence {
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
func (s *InitTreeStands) Update(_ *ecs.World) {}

// Finalize the system.
func (s *InitTreeStands) Finalize(_ *ecs.World) {}
