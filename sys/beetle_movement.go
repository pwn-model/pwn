package sys

import (
	"math/rand/v2"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/util"
)

func init() {
	config.Register[BeetleMovement]()
}

// BeetleMovement system.
type BeetleMovement struct {
	StepsPerTick          int     `yaml:"steps_per_tick"`
	DurationFeeding       int     `yaml:"duration_feeding"`
	DurationEggLaying     int     `yaml:"duration_egg_laying"`
	LeaveTreeProbability  float64 `yaml:"leave_tree_probability"`
	RandomWalkProbability float64 `yaml:"random_walk_probability"`

	timeRes         ecs.Resource[res.Time]
	randRes         ecs.Resource[resource.Rand]
	healthyFieldRes ecs.Resource[res.HealthyTreeAttraction]
	damagedFieldRes ecs.Resource[res.DamagedTreeAttraction]

	filter        *ecs.Filter2[comp.BeetlePosition, comp.EmergenceTick]
	filterHealthy *ecs.Filter1[comp.Position]
	filterDamaged *ecs.Filter1[comp.Position]

	healthyPresence res.Grid[bool]
	damagedPresence res.Grid[bool]

	width  int
	height int
}

// neighborOffsets are the (dx, dy) offsets of the 8 Moore-neighborhood cells
// around a beetle's current cell.
var neighborOffsets = [8][2]int{
	{-1, -1}, {0, -1}, {1, -1},
	{-1, 0}, {1, 0},
	{-1, 1}, {0, 1}, {1, 1},
}

// Initialize the system.
func (s *BeetleMovement) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)
	s.randRes = s.randRes.New(world)
	s.healthyFieldRes = s.healthyFieldRes.New(world)
	s.damagedFieldRes = s.damagedFieldRes.New(world)

	s.filter = s.filter.New(world)
	s.filterHealthy = s.filterHealthy.New(world).Without(ecs.C[comp.Damaged]())
	s.filterDamaged = s.filterDamaged.New(world).With(ecs.C[comp.Damaged]())

	ws := ecs.GetResource[res.WorldSize](world)
	s.healthyPresence = res.NewGrid[bool](ws.Width(), ws.Height(), ws.CellSize())
	s.damagedPresence = res.NewGrid[bool](ws.Width(), ws.Height(), ws.CellSize())
	s.width, s.height = ws.Width(), ws.Height()
}

// Update the system.
func (s *BeetleMovement) Update(world *ecs.World) {
	tick := s.timeRes.Get().Tick
	src := s.randRes.Get()
	rng := rand.New(src)
	healthyField := s.healthyFieldRes.Get().Grid
	damagedField := s.damagedFieldRes.Get().Grid
	var field res.Grid[float64]
	var presence res.Grid[bool]

	presenceCalculated := false

	q := s.filter.Query()
	for q.NextTable() {
		if !presenceCalculated {
			s.calcPresence(world)
			presenceCalculated = true
		}

		positions, emergenceTicks := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			et := &emergenceTicks[i]

			if tick < et.TickOfEmergence+s.DurationFeeding {
				field = healthyField
				presence = s.healthyPresence
			} else if tick < et.TickOfEmergence+s.DurationFeeding+s.DurationEggLaying {
				field = damagedField
				presence = s.damagedPresence
			} else {
				continue // TODO: remove beetle?
			}

			for range s.StepsPerTick {
				treeHere := presence.Get(pos.X, pos.Y)
				if !treeHere || rng.Float64() >= s.LeaveTreeProbability {
					if rng.Float64() < s.RandomWalkProbability {
						pos.X, pos.Y = s.randomNeighbor(src, pos.X, pos.Y)
					} else {
						pos.X, pos.Y = s.maxFieldNeighbor(&field, pos.X, pos.Y)
					}
					treeHere = presence.Get(pos.X, pos.Y)
				}

				if !treeHere {
					continue
				}

				// TODO: something with the tree...
			}
		}
	}
}

// randomNeighbor picks a uniformly random in-bounds Moore-neighborhood cell
// of (x, y), by rejection sampling: interior cells (the common case) resolve
// in a single draw, and only edge/corner cells ever redraw.
//
// Draws the offset index via util.RandRange rather than (*rand.Rand).IntN:
// IntN(8) would take stdlib's power-of-two fast path (a single masked draw,
// no rejection sampling), which diverges from the sibling Julia
// implementation's frozen_rand_range -- see util.RandRange's doc comment.
func (s *BeetleMovement) randomNeighbor(src rand.Source, x, y int) (int, int) {
	for {
		o := neighborOffsets[util.RandRange(src, uint64(len(neighborOffsets)))]
		nx, ny := x+o[0], y+o[1]
		if nx >= 0 && nx < s.width && ny >= 0 && ny < s.height {
			return nx, ny
		}
	}
}

// maxFieldNeighbor picks the in-bounds Moore-neighborhood cell of (x, y)
// with the highest value in field, breaking ties by neighborOffsets order.
func (s *BeetleMovement) maxFieldNeighbor(field *res.Grid[float64], x, y int) (int, int) {
	bestX, bestY := x, y
	hasBest := false
	var bestV float64
	for _, o := range neighborOffsets {
		nx, ny := x+o[0], y+o[1]
		if nx < 0 || nx >= s.width || ny < 0 || ny >= s.height {
			continue
		}
		if v := field.Get(nx, ny); !hasBest || v > bestV {
			bestV = v
			bestX, bestY = nx, ny
			hasBest = true
		}
	}
	return bestX, bestY
}

// Finalize the system.
func (s *BeetleMovement) Finalize(_ *ecs.World) {}

func (s *BeetleMovement) calcPresence(_ *ecs.World) {
	s.healthyPresence.Fill(false)
	s.damagedPresence.Fill(false)

	q := s.filterHealthy.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			s.healthyPresence.Set(pos.X, pos.Y, true)
		}
	}

	q = s.filterDamaged.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			s.damagedPresence.Set(pos.X, pos.Y, true)
		}
	}
}
