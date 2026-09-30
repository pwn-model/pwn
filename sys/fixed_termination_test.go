package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

func TestFixedTermination(t *testing.T) {
	a := app.New()

	s := FixedTermination{Steps: 10}
	s.Initialize(a.World)

	tick := ecs.GetResource[resource.Tick](a.World)
	term := ecs.GetResource[resource.Termination](a.World)

	tick.Tick = 8
	s.Update(a.World)
	assert.False(t, term.Terminate)

	// The 10th tick (0-based tick 9) is the last one.
	tick.Tick = 9
	s.Update(a.World)
	assert.True(t, term.Terminate)
}
