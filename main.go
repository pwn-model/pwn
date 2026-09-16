package main

import (
	"time"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark-tools/system"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/sys"
)

func main() {
	app := app.New()
	app.TPS = 0

	rnd := ecs.GetResource[resource.Rand](app.World)
	rnd.Source = res.NewXoshiro256pp(uint64(time.Now().UnixNano()))

	worldSize := res.WorldSize{
		Width:      1000,
		Height:     1000,
		Resolution: 50,
	}
	ecs.AddResource(app.World, &worldSize)

	app.AddSystem(&sys.InitGrids{})
	app.AddSystem(&sys.InitTrees{
		TreeProbability:  0.9,
		DamagePrevalence: 0.03,
	})
	app.AddSystem(&system.FixedTermination{Steps: 100})

	app.Run()
}
