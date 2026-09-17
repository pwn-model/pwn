package obs

import (
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
)

// TreeDamage reports the share of damaged trees.
type TreeDamage struct {
	total           *ecs.Filter0
	damaged         *ecs.Filter0
	infected        *ecs.Filter0
	damagedInfected *ecs.Filter0

	result []float64
}

// Initialize the observer.
func (o *TreeDamage) Initialize(w *ecs.World) {
	o.total = o.total.New(w).With(ecs.C[comp.Position]())
	o.damaged = o.damaged.New(w).With(ecs.C[comp.Damaged]())

	o.result = make([]float64, 1)
}

// Update the observer.
func (o *TreeDamage) Update(_ *ecs.World) {}

// Header returns the column names, in the same order as the values returned by Values.
func (o *TreeDamage) Header() []string {
	return []string{"damaged"}
}

// Values for the current model tick, in the order given by Header.
func (o *TreeDamage) Values(_ *ecs.World) []float64 {
	o.result[0] = float64(count(o.damaged)) / float64(count(o.total))

	return o.result
}
