package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func TestUpdateTime(t *testing.T) {
	a := app.New()

	s := UpdateTime{TicksPerYear: 52}
	s.Initialize(a.World)

	tick := ecs.GetResource[resource.Tick](a.World)
	time := ecs.GetResource[res.Time](a.World)

	tick.Tick = 0
	s.Update(a.World)
	assert.Equal(t, 0, time.Tick)
	assert.Equal(t, 0, time.TickOfYear)
	assert.Equal(t, 0, time.Year)

	// Somewhere in the middle of the first year.
	tick.Tick = 10
	s.Update(a.World)
	assert.Equal(t, 10, time.Tick)
	assert.Equal(t, 10, time.TickOfYear)
	assert.Equal(t, 0, time.Year)

	// Exactly one full year elapsed: wraps into year 1, week 0.
	tick.Tick = 52
	s.Update(a.World)
	assert.Equal(t, 52, time.Tick)
	assert.Equal(t, 0, time.TickOfYear)
	assert.Equal(t, 1, time.Year)

	// Partway through the second year.
	tick.Tick = 60
	s.Update(a.World)
	assert.Equal(t, 60, time.Tick)
	assert.Equal(t, 8, time.TickOfYear)
	assert.Equal(t, 1, time.Year)
}
