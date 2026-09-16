package res

import (
	"math/rand/v2"
	"testing"

	"github.com/mlange-42/ark-tools/resource"
)

// BenchmarkFloat64PCG benchmarks resource.Rand's original default source,
// the PCG-DXSM generator from math/rand/v2 (superseded in the model itself
// by Xoshiro256pp, see main.go), for comparison against BenchmarkFloat64Xoshiro.
func BenchmarkFloat64PCG(b *testing.B) {
	r := rand.New(resource.Rand{Source: rand.NewPCG(0, 1)})

	b.ResetTimer()
	for b.Loop() {
		_ = r.Float64()
	}
}

// BenchmarkFloat64Xoshiro benchmarks Xoshiro256pp, a port of Julia's default
// RNG and the source now used for the model's resource.Rand (see main.go).
func BenchmarkFloat64Xoshiro(b *testing.B) {
	r := rand.New(NewXoshiro256pp(1))

	b.ResetTimer()
	for b.Loop() {
		_ = r.Float64()
	}
}
