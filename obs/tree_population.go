package obs

import (
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
)

// TreePopulationObserver reports total and damaged tree counts per model tick.
type TreePopulationObserver struct {
	total           *ecs.Filter0
	damaged         *ecs.Filter0
	infected        *ecs.Filter0
	damagedInfected *ecs.Filter0

	result []float64
}

// Initialize the observer.
func (o *TreePopulationObserver) Initialize(w *ecs.World) {
	o.total = o.total.New(w).With(ecs.C[comp.Position]())
	o.damaged = o.damaged.New(w).With(ecs.C[comp.Damaged]()).Without(ecs.C[comp.Infected]())
	o.infected = o.infected.New(w).With(ecs.C[comp.Infected]()).Without(ecs.C[comp.Damaged]())
	o.damagedInfected = o.damagedInfected.New(w).With(ecs.C[comp.Damaged](), ecs.C[comp.Infected]())

	o.result = make([]float64, 4)
}

// Update the observer.
func (o *TreePopulationObserver) Update(_ *ecs.World) {}

// Header returns the column names, in the same order as the values returned by Values.
func (o *TreePopulationObserver) Header() []string {
	return []string{"total", "damaged", "infected", "damaged_infected"}
}

// Values for the current model tick, in the order given by Header.
func (o *TreePopulationObserver) Values(_ *ecs.World) []float64 {
	o.result[0] = float64(count(o.total))
	o.result[1] = float64(count(o.damaged))
	o.result[2] = float64(count(o.infected))
	o.result[3] = float64(count(o.damagedInfected))

	return o.result
}

func count(f *ecs.Filter0) int {
	q := f.Query()
	n := q.Count()
	q.Close()
	return n
}
