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

// BeetleMovement system.
type BeetleMovement struct {
	StepsPerTick    int     `yaml:"steps_per_tick"`
	DistancePerStep float64 `yaml:"distance_per_step"`
	// EmergenceFlightDistance is the maximum distance, in meters, of the
	// undirected flight that beetles make in their first movement step after
	// emergence, before flying towards a target. The actual distance is
	// drawn uniformly between 0 and this value. Spreads out the beetles
	// emerging from the same tree, which would otherwise all fly to the same
	// first target.
	EmergenceFlightDistance float64 `yaml:"emergence_flight_distance"`
	DurationFeeding         int     `yaml:"duration_feeding"`
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

	// toRemove is a reused buffer for beetles that left the world.
	toRemove []ecs.Entity

	distanceInCells        float64
	emergenceFlightInCells float64
	width                  int
	height                 int
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
	s.emergenceFlightInCells = s.EmergenceFlightDistance / float64(ws.CellSize())
}

// Update the system.
func (s *BeetleMovement) Update(world *ecs.World) {
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
		entities := q.Entities()
		positions, emergenceTicks := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			et := &emergenceTicks[i]

			for step := range s.StepsPerTick {
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
				inside := true
				if step == 0 && tick == et.TickOfEmergence {
					// Undirected flight away from the tree the beetle emerged from,
					// in one go, in a uniformly random direction.
					flight := min(rng.Float64()*s.emergenceFlightInCells, maxStep)
					dx, dy := util.RandomDirection(rng)
					pos.X += int(math.Round(dx * flight))
					pos.Y += int(math.Round(dy * flight))
					maxStep -= flight
					inside = nearest.Contains(pos.X, pos.Y)
				} else if presence.Get(pos.X, pos.Y) && rng.Float64() < s.LeaveTreeProbability {
					// Walk out of the area attracted by the tree the beetle leaves.
					maxStep, inside = walk(rng, pos, &nearest, maxStep)
				}

				if !inside {
					// The beetle left the world and dies.
					s.toRemove = append(s.toRemove, entities[i])
					break
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

	for _, e := range s.toRemove {
		world.RemoveEntity(e)
	}
	s.toRemove = s.toRemove[:0]
}

// Finalize the system.
func (s *BeetleMovement) Finalize(_ *ecs.World) {}

// feed counts a beetle feeding in cell (x, y).
func (s *BeetleMovement) feed(x, y int) {
	s.feedingInfected.Set(x, y, s.feedingInfected.Get(x, y)+1)
}

// walk moves pos along a straight line in a uniformly random direction, out
// of the area of the source the beetle is in. It advances in steps of one
// cell size of distance, each costing 1 of the budget, and stops as soon as
// the cell's source differs, or the remaining budget (in cells) is below 1.
// Returns the remaining budget, and false if the beetle left the grid (pos
// is then outside of it).
func walk(rng *rand.Rand, pos *comp.BeetlePosition, source *res.Grid[uint32], budget float64) (float64, bool) {
	leaving := source.Get(pos.X, pos.Y)
	dx, dy := util.RandomDirection(rng)
	x0, y0 := pos.X, pos.Y
	for k := 1; budget >= 1 && source.Get(pos.X, pos.Y) == leaving; k++ {
		pos.X = x0 + int(math.Round(float64(k)*dx))
		pos.Y = y0 + int(math.Round(float64(k)*dy))
		budget--
		if !source.Contains(pos.X, pos.Y) {
			return budget, false
		}
	}
	return budget, true
}
