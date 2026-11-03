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

	filter *ecs.Filter2[comp.BeetlePosition, comp.EmergenceTick]

	feedingInfected res.Grid[uint32]

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

	ws := ecs.GetResource[res.WorldSize](world)
	s.width, s.height = ws.Width(), ws.Height()
	s.feedingInfected = res.NewGrid[uint32](ws.Width(), ws.Height(), ws.CellSize())

	ecs.AddResource(world, &res.FeedingInfectedBeetles{Grid: s.feedingInfected})
}

// Update the system.
func (s *BeetleMovement) Update(world *ecs.World) {
	tick := s.timeRes.Get().Tick
	src := s.randRes.Get()
	rng := rand.New(src)

	healthy := s.healthyFieldRes.Get()
	damaged := s.damagedFieldRes.Get()
	var field res.Grid[float64]
	var presence res.Grid[bool]

	s.feedingInfected.Fill(0)

	q := s.filter.Query()
	for q.NextTable() {
		positions, emergenceTicks := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			et := &emergenceTicks[i]

			switch {
			case tick < et.TickOfEmergence+s.DurationFeeding:
				field = healthy.Attraction
				presence = healthy.Presence
			case tick < et.TickOfEmergence+s.DurationFeeding+s.DurationEggLaying:
				field = damaged.Attraction
				presence = damaged.Presence
			default:
				continue // TODO: remove beetle?
			}

			for range s.StepsPerTick {
				if !presence.Get(pos.X, pos.Y) || rng.Float64() < s.LeaveTreeProbability {
					if rng.Float64() < s.RandomWalkProbability {
						pos.X, pos.Y = s.randomNeighbor(src, pos.X, pos.Y)
					} else {
						pos.X, pos.Y = s.maxFieldNeighbor(src, &field, pos.X, pos.Y)
					}
				}

				if !presence.Get(pos.X, pos.Y) {
					continue
				}

				feeding := s.feedingInfected.Get(pos.X, pos.Y)
				s.feedingInfected.Set(pos.X, pos.Y, feeding+1)
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
// with the highest value in field. Ties are broken uniformly at random, so
// that a flat field doesn't bias movement towards the first neighborOffsets.
// Draws a random number only if there is a tie, via util.RandRange (see
// randomNeighbor).
func (s *BeetleMovement) maxFieldNeighbor(src rand.Source, field *res.Grid[float64], x, y int) (int, int) {
	var candidates [len(neighborOffsets)]uint8
	n := 0
	var bestV float64
	for i, o := range neighborOffsets {
		nx, ny := x+o[0], y+o[1]
		if nx < 0 || nx >= s.width || ny < 0 || ny >= s.height {
			continue
		}
		v := field.Get(nx, ny)
		if n == 0 || v > bestV {
			bestV = v
			candidates[0] = uint8(i)
			n = 1
		} else if v == bestV {
			candidates[n] = uint8(i)
			n++
		}
	}

	switch n {
	case 0:
		return x, y
	case 1:
		o := neighborOffsets[candidates[0]]
		return x + o[0], y + o[1]
	default:
		o := neighborOffsets[candidates[util.RandRange(src, uint64(n))]]
		return x + o[0], y + o[1]
	}
}

// Finalize the system.
func (s *BeetleMovement) Finalize(_ *ecs.World) {}
