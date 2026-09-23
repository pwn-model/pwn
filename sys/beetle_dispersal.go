package sys

import (
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[BeetleDispersal]()
}

// BeetleDispersal system.
type BeetleDispersal struct {
	TickOfYear        int `yaml:"tick_of_year"`        // Tick of year when tree attraction is calculated.
	CellSize          int `yaml:"cell_size"`           // Cell size of the attraction grid, in meters.
	EggLayingStart    int `yaml:"egg_laying_start"`    // Age at which beetles start laying eggs, in weeks.
	EggLayingDuration int `yaml:"egg_laying_duration"` // Maximum duration of egg laying, in weeks.

	timeRes ecs.Resource[res.Time]

	filterHealthy *ecs.Filter1[comp.Position]
	filterDamaged *ecs.Filter1[comp.Position]

	// unitsPerCell is the number of tree-grid units per dispersal-grid
	// cell, i.e. CellSize expressed in the world's base cell-size units.
	unitsPerCell int

	nearestHealthy res.Grid[float64]
	nearestDamaged res.Grid[float64]
}

// Initialize the system.
func (s *BeetleDispersal) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)

	s.filterHealthy = s.filterHealthy.New(world).Without(ecs.C[comp.Damaged]())
	s.filterDamaged = s.filterDamaged.New(world).With(ecs.C[comp.Damaged]())

	ws := ecs.GetResource[res.WorldSize](world)
	if s.CellSize%ws.CellSize() != 0 {
		panic("CellSize of the dispersal submodel must be a multiple of the world's base cell size.")
	}
	s.unitsPerCell = s.CellSize / ws.CellSize()

	s.nearestHealthy = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	s.nearestDamaged = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
}

// Update the system.
func (s *BeetleDispersal) Update(world *ecs.World) {
	toy := s.timeRes.Get().TickOfYear

	if toy == s.TickOfYear {
		s.calcAttraction()
	}
}

// Finalize the system.
func (s *BeetleDispersal) Finalize(_ *ecs.World) {}

func (s *BeetleDispersal) calcAttraction() {
	s.fillFromQuery(&s.nearestHealthy, s.filterHealthy)
	s.fillFromQuery(&s.nearestDamaged, s.filterDamaged)

	s.fillGrid(&s.nearestHealthy, 5)
	s.fillGrid(&s.nearestDamaged, 5)
}

func (s *BeetleDispersal) fillFromQuery(grid *res.Grid[float64], filter *ecs.Filter1[comp.Position]) {
	grid.Fill(0.0)
	q := filter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			grid.Set(pos.X, pos.Y, 1.0)
		}
	}
}

func (s *BeetleDispersal) fillGrid(grid *res.Grid[float64], radius int) {

}
