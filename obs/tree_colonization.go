package obs

import (
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
)

// TreeColonization reports the share of colonized trees.
type TreeColonization struct {
	total     *ecs.Filter0
	colonized *ecs.Filter0

	result []float64
}

// Initialize the observer.
func (o *TreeColonization) Initialize(w *ecs.World) {
	o.total = o.total.New(w).With(ecs.C[comp.Position]())
	o.colonized = o.colonized.New(w).With(ecs.C[comp.Colonized]())

	o.result = make([]float64, 1)
}

// Update the observer.
func (o *TreeColonization) Update(_ *ecs.World) {}

// Header returns the column names, in the same order as the values returned by Values.
func (o *TreeColonization) Header() []string {
	return []string{"colonized"}
}

// Values for the current model tick, in the order given by Header.
func (o *TreeColonization) Values(_ *ecs.World) []float64 {
	o.result[0] = float64(count(o.colonized)) / float64(count(o.total))

	return o.result
}
