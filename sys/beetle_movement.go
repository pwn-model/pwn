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
	config.Register[BeetleMovement]()
}

// BeetleMovement system.
type BeetleMovement struct {
	StepsPerTick          int     `yaml:"steps_per_tick"`
	DistancePerStep       float64 `yaml:"distance_per_step"`
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

	maxStep := s.distanceInCells
	maxStepSq := maxStep * maxStep

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

			switch {
			case tick < et.TickOfEmergence+s.DurationFeeding:
				nearest = healthy.Source
				presence = healthy.Presence
			case tick < et.TickOfEmergence+s.DurationFeeding+s.DurationEggLaying:
				nearest = damaged.Source
				presence = damaged.Presence
			default:
				continue // TODO: remove beetle?
			}

			for range s.StepsPerTick {
				if !presence.Get(pos.X, pos.Y) || rng.Float64() < s.LeaveTreeProbability {
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

					/*if rng.Float64() < s.RandomWalkProbability {
						pos.X, pos.Y = s.randomNeighbor(src, pos.X, pos.Y)
					} else {
						pos.X, pos.Y = s.maxFieldNeighbor(src, &field, pos.X, pos.Y)
					}*/
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

// Finalize the system.
func (s *BeetleMovement) Finalize(_ *ecs.World) {}
