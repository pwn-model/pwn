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
	config.Register[BeetleMovementOctants]()
}

// BeetleMovementOctants system.
//
// Like [BeetleMovement], but instead of flying towards the single
// attraction source of their current cell, beetles select one of the 8
// octants around their cell by roulette selection, weighted by the
// attraction values of the octants' sources (see [TreeAttractionOctants]),
// and fly straight towards that octant's source, by up to DistancePerStep
// meters per step.
//
// The selection is memoryless: a beetle that does not reach its target
// within a step selects anew, from the octants of the cell it got to, in
// its next step.
//
// A beetle leaving a tree selects among the octants of the tree's cell
// right away, without first walking out of the tree's area as in
// BeetleMovement: a cell's own tree is never one of its octant sources, so
// the other trees around it are available directly.
//
// A beetle whose cell has no source in any octant (no tree of the wanted
// type visible) stays where it is, without drawing a random number.
//
// Mirrors the sibling Julia implementation's BeetleMovementOctants.
type BeetleMovementOctants struct {
	StepsPerTick    int     `yaml:"steps_per_tick"`
	DistancePerStep float64 `yaml:"distance_per_step"`
	// EmergenceFlightDistance is the maximum distance, in meters, of the
	// undirected flight that beetles make in their first movement step after
	// emergence, before flying towards a target. The actual distance is
	// drawn uniformly between 0 and this value. See BeetleMovement.
	EmergenceFlightDistance float64 `yaml:"emergence_flight_distance"`
	DurationFeeding         int     `yaml:"duration_feeding"`
	// FeedingProbability is the probability that a beetle past
	// DurationFeeding feeds (targets healthy trees) instead of laying eggs
	// (targets damaged trees), drawn anew each movement step.
	FeedingProbability   float64 `yaml:"feeding_probability"`
	LeaveTreeProbability float64 `yaml:"leave_tree_probability"`

	timeRes    ecs.Resource[res.Time]
	randRes    ecs.Resource[resource.Rand]
	healthyRes ecs.Resource[res.HealthyTreeAttractionOctants]
	damagedRes ecs.Resource[res.DamagedTreeAttractionOctants]

	filter *ecs.Filter2[comp.BeetlePosition, comp.EmergenceTick]

	feedingInfected res.Grid[uint32]

	// toRemove is a reused buffer for beetles that left the world.
	toRemove []ecs.Entity

	distanceInCells        float64
	emergenceFlightInCells float64
}

// Initialize the system.
func (s *BeetleMovementOctants) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)
	s.randRes = s.randRes.New(world)
	s.healthyRes = s.healthyRes.New(world)
	s.damagedRes = s.damagedRes.New(world)

	s.filter = s.filter.New(world)

	ws := ecs.GetResource[res.WorldSize](world)
	s.feedingInfected = res.NewGrid[uint32](ws.Width(), ws.Height(), ws.CellSize())

	ecs.AddResource(world, &res.FeedingInfectedBeetles{Grid: s.feedingInfected})

	s.distanceInCells = s.DistancePerStep / float64(ws.CellSize())
	s.emergenceFlightInCells = s.EmergenceFlightDistance / float64(ws.CellSize())
}

// Update the system.
func (s *BeetleMovementOctants) Update(world *ecs.World) {
	tick := s.timeRes.Get().Tick
	src := s.randRes.Get()
	rng := rand.New(src)

	healthy := s.healthyRes.Get()
	damaged := s.damagedRes.Get()
	var sources *res.Grid[res.OctantSources]
	var presence *res.Grid[bool]

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
					sources, presence = &healthy.Sources, &healthy.Presence
				} else {
					sources, presence = &damaged.Sources, &damaged.Presence
				}

				maxStep := s.distanceInCells
				inside := true
				leaving := false
				if step == 0 && tick == et.TickOfEmergence {
					// Undirected flight away from the tree the beetle emerged from,
					// in one go, in a uniformly random direction. Without any
					// emergence flight, no random numbers are drawn for it.
					if s.emergenceFlightInCells > 0 {
						flight := min(rng.Float64()*s.emergenceFlightInCells, maxStep)
						dx, dy := util.RandomDirection(rng)
						pos.X += int(math.Round(dx * flight))
						pos.Y += int(math.Round(dy * flight))
						maxStep -= flight
						inside = presence.Contains(pos.X, pos.Y)
					}
				} else if presence.Get(pos.X, pos.Y) && rng.Float64() < s.LeaveTreeProbability {
					leaving = true
				}

				if !inside {
					// The beetle left the world and dies.
					s.toRemove = append(s.toRemove, entities[i])
					break
				}

				if presence.Get(pos.X, pos.Y) && !leaving {
					s.feed(pos.X, pos.Y)
					continue
				}

				r := sources.Ptr(pos.X, pos.Y)
				k := selectOctant(rng, r)
				if k < 0 {
					// No tree visible: stay in place.
					if presence.Get(pos.X, pos.Y) {
						s.feed(pos.X, pos.Y)
					}
					continue
				}

				o := r.Offsets[k]
				dx, dy := float64(o[0]), float64(o[1])
				distSq := dx*dx + dy*dy
				if distSq <= maxStep*maxStep {
					pos.X += int(o[0])
					pos.Y += int(o[1])
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
func (s *BeetleMovementOctants) Finalize(_ *ecs.World) {}

// feed counts a beetle feeding in cell (x, y).
func (s *BeetleMovementOctants) feed(x, y int) {
	s.feedingInfected.Set(x, y, s.feedingInfected.Get(x, y)+1)
}

// selectOctant selects one of the octants of r by roulette selection, with
// probabilities proportional to the octants' attraction values. Returns the
// selected octant, or -1 without drawing a random number if all octants
// are empty.
//
// The attraction values are summed as float64, in octant order, both for
// the total and the running sum the draw is compared against, so the final
// running sum equals the total exactly. A draw that rounds up to the total
// selects the last non-empty octant.
//
// Mirrors the sibling Julia implementation's select_octant, whose octants
// are 1-based and which returns 0 instead of -1.
func selectOctant(rng *rand.Rand, r *res.OctantSources) int {
	total := 0.0
	for _, a := range r.Attraction {
		total += float64(a)
	}
	if total <= 0 {
		return -1
	}

	u := rng.Float64() * total
	acc := 0.0
	last := -1
	for k, a32 := range r.Attraction {
		a := float64(a32)
		if a > 0 {
			acc += a
			if u < acc {
				return k
			}
			last = k
		}
	}
	return last
}
