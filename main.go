package main

import (
	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/system"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/sys"
)

func main() {
	app := app.New()
	app.TPS = 0

	treeGrid := res.NewEntityGrid(100, 100)
	ecs.AddResource(app.World, &treeGrid)

	app.AddSystem(&sys.InitTrees{TreeProbability: 0.9})
	app.AddSystem(&system.FixedTermination{Steps: 100})

	app.Run()
}
