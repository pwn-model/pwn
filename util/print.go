package util

import (
	"strings"

	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
)

// TreesToString prints a tree grid.
func TreesToString(world *ecs.World) string {
	grid := ecs.GetResource[res.EntityGrid](world)
	infMap := ecs.NewMap[comp.Infected](world)
	dmgMap := ecs.NewMap[comp.Damaged](world)

	b := strings.Builder{}

	for x := range grid.Width() {
		for y := range grid.Height() {
			e := grid.Get(x, y)
			if e.IsZero() {
				b.WriteRune(' ')
				continue
			}

			if infMap.Has(e) {
				b.WriteRune('x')
				continue
			}

			if dmgMap.Has(e) {
				b.WriteRune('+')
				continue
			}

			b.WriteRune('.')
		}
		b.WriteRune('\n')
	}

	return b.String()
}
