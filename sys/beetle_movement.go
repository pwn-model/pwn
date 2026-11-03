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
	config.Register[BeetleMovement]()
}

// neighborOffsets are the (dx, dy) offsets of the 8 Moore-neighborhood cells
// around a beetle's current cell.
var neighborOffsets = [8][2]int{
	{-1, -1}, {0, -1}, {1, -1},
	{-1, 0}, {1, 0},
	{-1, 1}, {0, 1}, {1, 1},
}

// BeetleMovement system.
type BeetleMovement struct {
	StepsPerTick    int     `yaml:"steps_per_tick"`
	DistancePerStep float64 `yaml:"distance_per_step"`
	DurationFeeding int     `yaml:"duration_feeding"`
	// FeedingProbability is the probability that a beetle past
	// DurationFeeding feeds (targets healthy trees) instead of laying eggs
	// (targets damaged trees), drawn anew each movement step.
	FeedingProbability   float64 `yaml:"feeding_probability"`
	LeaveTreeProbability float64 `yaml:"leave_tree_probability"`

	timeRes         ecs.Resource[res.Time]
	randRes         ecs.Resource[resource.Rand]
	healthyFieldRes ecs.Resource[res.HealthyTreeAttraction]
	damagedFieldRes ecs.Resource[res.DamagedTreeAttraction]

	filter *ecs.Filter2[comp.BeetlePosition, comp.EmergenceTick]

	feedingInfected res.Grid[uint32]

	distanceInCells float64
	width           int
	height          int
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

	s.distanceInCells = s.DistancePerStep / float64(ws.CellSize())
}

// Update the system.
func (s *BeetleMovement) Update(_ *ecs.World) {
	tick := s.timeRes.Get().Tick
	src := s.randRes.Get()
	rng := rand.New(src)

	healthy := s.healthyFieldRes.Get()
	damaged := s.damagedFieldRes.Get()
	var nearest res.Grid[uint32]
	var presence res.Grid[bool]

	s.feedingInfected.Fill(0)

	q := s.filter.Query()
	for q.NextTable() {
		positions, emergenceTicks := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			et := &emergenceTicks[i]

			for range s.StepsPerTick {
				isFeeding := true
				if tick >= et.TickOfEmergence+s.DurationFeeding {
					isFeeding = rng.Float64() < s.FeedingProbability
				}

				if isFeeding {
					nearest = healthy.Source
					presence = healthy.Presence
				} else {
					nearest = damaged.Source
					presence = damaged.Presence
				}

				maxStep := s.distanceInCells
				if presence.Get(pos.X, pos.Y) && rng.Float64() < s.LeaveTreeProbability {
					lastTree := nearest.Get(pos.X, pos.Y)
					offset := randomOffset(src)
					for nearest.Get(pos.X, pos.Y) == lastTree && maxStep >= 1 {
						pos.X += offset[0]
						pos.Y += offset[1]
						maxStep -= 1 // TODO: exact cost for diagonal steps?
						if !presence.Contains(pos.X, pos.Y) {
							pos.X -= offset[0]
							pos.Y -= offset[1]
							offset = randomOffset(src)
							continue
						}
					}
				}

				if presence.Get(pos.X, pos.Y) {
					s.feed(pos.X, pos.Y)
					continue
				}

				maxStepSq := maxStep * maxStep

				tx, ty := nearest.Coords(nearest.Get(pos.X, pos.Y))
				dx, dy := float64(tx-pos.X), float64(ty-pos.Y)
				distSq := dx*dx + dy*dy
				if distSq <= maxStepSq {
					pos.X, pos.Y = tx, ty
				} else {
					frac := maxStep / math.Sqrt(distSq)
					pos.X += int(math.Round(dx * frac))
					pos.Y += int(math.Round(dy * frac))
				}

				if presence.Get(pos.X, pos.Y) {
					s.feed(pos.X, pos.Y)
				}
			}
		}
	}
}

// Finalize the system.
func (s *BeetleMovement) Finalize(_ *ecs.World) {}

// feed counts a beetle feeding in cell (x, y).
func (s *BeetleMovement) feed(x, y int) {
	s.feedingInfected.Set(x, y, s.feedingInfected.Get(x, y)+1)
}

// randomOffset draws one of the neighborOffsets uniformly at random, via
// util.RandRange (see BeetleMovementByField.randomNeighbor).
func randomOffset(src rand.Source) [2]int {
	return neighborOffsets[util.RandRange(src, uint64(len(neighborOffsets)))]
}
