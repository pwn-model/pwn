package main

import (
	"flag"
	"os"
	"testing"

	"github.com/pwn-model/pwn/benchmark"
)

func main() {
	testing.Init()
	flag.Parse()

	repetitions := 1

	f, err := os.Create("bench.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	formats := []benchmark.Format{
		{Format: benchmark.ToCSV, Writer: f},
	}

	benchmark.RunBenchmarks("Compare", benchesCompare(), repetitions, formats)
}

func benchesCompare() []benchmark.Benchmark {
	return []benchmark.Benchmark{
		{Name: "Setup model", Desc: "", F: setupOnly, N: 1},
		{Name: "Run model", Desc: "", F: runOnly, N: 1},
		{Name: "Setup + run model", Desc: "", F: setupAndRun, N: 1},
	}
}
