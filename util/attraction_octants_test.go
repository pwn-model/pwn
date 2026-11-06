package util

import (
	"math"
	"math/rand/v2"
	"testing"
	"unsafe"

	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

// inOctant reports whether offset (dx, dy) lies in the half-open octant k
// (see res.OctantSources).
func inOctant(k, dx, dy int) bool {
	switch k {
	case 0:
		return 0 <= dy && dy < dx
	case 1:
		return 0 < dx && dx <= dy
	case 2:
		return 0 <= -dx && -dx < dy
	case 3:
		return 0 < dy && dy <= -dx
	case 4:
		return 0 <= -dy && -dy < -dx
	case 5:
		return 0 < -dx && -dx <= -dy
	case 6:
		return 0 <= dx && dx < -dy
	case 7:
		return 0 < -dy && -dy <= dx
	}
	panic("invalid octant")
}

// octantOf returns the half-open octant of offset (dx, dy) from its angle,
// independently of inOctant.
func octantOf(dx, dy int) int {
	a := math.Atan2(float64(dy), float64(dx))
	if a < 0 {
		a += 2 * math.Pi
	}
	return int(math.Floor(a / (math.Pi / 4)))
}

// octantAttraction is the attraction of a source with the given seed at
// offset (dx, dy), evaluated directly from the chamfer path: max(|dx|, |dy|)
// - min(|dx|, |dy|) axis steps and min(|dx|, |dy|) diagonal steps.
func octantAttraction(seed float64, dx, dy int, decay float64) float64 {
	ax, ay := abs(dx), abs(dy)
	lo, hi := min(ax, ay), max(ax, ay)
	return seed * math.Pow(decay, float64(hi-lo)) * math.Pow(math.Pow(decay, math.Sqrt2), float64(lo))
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func runOctants(seeds *res.Grid[float64], halfDistance float64) res.Grid[res.OctantSources] {
	sources := res.NewGrid[res.OctantSources](seeds.Width(), seeds.Height(), seeds.CellSize())
	buf := NewOctantBuffers(seeds.Height())
	PropagateOctants(&sources, seeds, halfDistance, &buf)
	return sources
}

func TestHalfOpenOctantsPartitionThePlane(t *testing.T) {
	for dx := -6; dx <= 6; dx++ {
		for dy := -6; dy <= 6; dy++ {
			n := 0
			for k := range 8 {
				if inOctant(k, dx, dy) {
					n++
				}
			}
			if dx == 0 && dy == 0 {
				assert.Equal(t, 0, n)
				continue
			}
			assert.Equal(t, 1, n, "offset (%d, %d)", dx, dy)
			// Rays exactly on a multiple of 45° are the only cases where
			// floating-point atan could round across a boundary; check them
			// by construction instead.
			if dx == 0 || dy == 0 || abs(dx) == abs(dy) {
				continue
			}
			assert.True(t, inOctant(octantOf(dx, dy), dx, dy), "offset (%d, %d)", dx, dy)
		}
	}
}

func TestOctantSourcesIsOneCacheLine(t *testing.T) {
	assert.Equal(t, uintptr(64), unsafe.Sizeof(res.OctantSources{}))
}

func TestPropagateOctantsSingleTree(t *testing.T) {
	seeds := res.NewGrid[float64](30, 20, 10)
	seeds.Set(14, 9, 1.0)
	halfDistance := 50.0
	sources := runOctants(&seeds, halfDistance)
	decay := math.Exp(-10.0 * math.Ln2 / halfDistance)

	// The tree's own cell has no source in any octant.
	assert.Equal(t, res.OctantSources{}, sources.Get(14, 9))

	// West of the tree, it lies on the +x axis, which belongs to octant 0
	// only; all other octants are empty.
	r := sources.Get(4, 9)
	for k := range 8 {
		if k == 0 {
			assert.Equal(t, [2]int16{10, 0}, r.Offsets[k])
			assert.Equal(t, float32(math.Pow(decay, 10)), r.Attraction[k])
		} else {
			assert.Equal(t, [2]int16{}, r.Offsets[k])
			assert.Equal(t, float32(0), r.Attraction[k])
		}
	}

	// At offset (3, 7) from the cell, the tree lies strictly inside octant 1.
	r = sources.Get(11, 2)
	for k := range 8 {
		if k == 1 {
			assert.Equal(t, [2]int16{3, 7}, r.Offsets[k])
			assert.InDelta(t, octantAttraction(1.0, 3, 7, decay), float64(r.Attraction[k]), 1e-6)
		} else {
			assert.Equal(t, [2]int16{}, r.Offsets[k])
		}
	}
}

func TestPropagateOctantsBoundaryRaysBelongToTheOctantTheyStart(t *testing.T) {
	// The ray at 45°*k belongs to octant k only.
	directions := [8][2]int{{1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}}
	for k, u := range directions {
		seeds := res.NewGrid[float64](30, 20, 10)
		seeds.Set(14+3*u[0], 9+3*u[1], 1.0)
		sources := runOctants(&seeds, 50.0)
		r := sources.Get(14, 9)
		for kk := range 8 {
			if kk == k {
				assert.Equal(t, [2]int16{int16(3 * u[0]), int16(3 * u[1])}, r.Offsets[kk])
				assert.Greater(t, r.Attraction[kk], float32(0))
			} else {
				assert.Equal(t, [2]int16{}, r.Offsets[kk])
				assert.Equal(t, float32(0), r.Attraction[kk])
			}
		}
	}
}

func TestPropagateOctantsFartherStrongerSourceIsNotShadowed(t *testing.T) {
	// A weak tree right in front of a strong one, on the same ray: the
	// strong tree must win where it is more attractive, although every path
	// from it passes the weak tree.
	seeds := res.NewGrid[float64](60, 5, 10)
	seeds.Set(19, 2, 0.1)
	seeds.Set(29, 2, 1.0)
	sources := runOctants(&seeds, 100.0) // factor 2 per 10 cells

	// From x=9, the weak tree is 10 cells away (0.1 / 2), the strong one 20
	// cells (1.0 / 4).
	assert.Equal(t, [2]int16{20, 0}, sources.Get(9, 2).Offsets[0])
	// Right next to the weak tree, it wins (0.1 * 0.93 vs. 1.0 / 2.1).
	assert.Equal(t, [2]int16{11, 0}, sources.Get(18, 2).Offsets[0])
	// From the weak tree's own cell, the strong tree is the source.
	assert.Equal(t, [2]int16{10, 0}, sources.Get(19, 2).Offsets[0])
}

func TestPropagateOctantsMatchesBruteForce(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	for _, size := range [][2]int{{23, 17}, {1, 9}, {9, 1}, {16, 16}} {
		w, h := size[0], size[1]
		seeds := res.NewGrid[float64](w, h, 10)
		var trees [][2]int
		for x := range w {
			for y := range h {
				corner := (x == 0 || x == w-1) && (y == 0 || y == h-1)
				if rng.Float64() < 0.08 || corner {
					seeds.Set(x, y, 0.05+0.95*rng.Float64())
					trees = append(trees, [2]int{x, y})
				}
			}
		}
		halfDistance := 40.0
		decay := math.Exp(-10.0 * math.Ln2 / halfDistance)
		sources := runOctants(&seeds, halfDistance)

		for x := range w {
			for y := range h {
				r := sources.Get(x, y)
				for k := range 8 {
					best := 0.0
					for _, tr := range trees {
						dx, dy := tr[0]-x, tr[1]-y
						if !inOctant(k, dx, dy) {
							continue
						}
						best = max(best, octantAttraction(seeds.Get(tr[0], tr[1]), dx, dy, decay))
					}

					ox, oy := int(r.Offsets[k][0]), int(r.Offsets[k][1])
					if best == 0 {
						assert.Equal(t, [2]int16{}, r.Offsets[k])
						assert.Equal(t, float32(0), r.Attraction[k])
						continue
					}
					// The source is a tree in the octant, and as attractive as
					// the best one (up to rounding of the different evaluation
					// order; random seeds make exact ties practically
					// impossible).
					assert.True(t, inOctant(k, ox, oy), "cell (%d, %d), octant %d", x, y, k)
					if !seeds.Contains(x+ox, y+oy) {
						t.Fatalf("source outside the grid: cell (%d, %d), octant %d", x, y, k)
					}
					seed := seeds.Get(x+ox, y+oy)
					assert.Greater(t, seed, 0.0)
					assert.InEpsilon(t, best, octantAttraction(seed, ox, oy, decay), 1e-12)
					assert.InEpsilon(t, float32(best), r.Attraction[k], 1e-6)
				}
			}
		}
	}
}
