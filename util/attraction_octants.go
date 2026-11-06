package util

import (
	"math"

	"github.com/pwn-model/pwn/res"
)

// OctantBuffers holds the reused scratch columns of PropagateOctants. See
// there.
type OctantBuffers struct {
	// values and offsets hold, in this order, the previous and current
	// column of the x-major octant, then the previous and current column of
	// the y-major octant of the quadrant being swept. Each has length
	// height+2: entry y+1 is grid row y, entries 0 and height+1 are padding
	// that always stays empty, standing in for rows outside the grid.
	values  [4][]float64
	offsets [4][][2]int16
}

// NewOctantBuffers creates OctantBuffers for grids of the given height.
func NewOctantBuffers(height int) OctantBuffers {
	var b OctantBuffers
	for i := range 4 {
		b.values[i] = make([]float64, height+2)
		b.offsets[i] = make([][2]int16, height+2)
	}
	return b
}

// PropagateOctants computes, for every cell and each of its 8 half-open
// octants (see [res.OctantSources]), the most attractive source in that
// octant, and stores its offset and attraction value in sources.
//
// A source's attraction, as seen from a cell, is its seed (from seeds; 0
// for cells without a source) decayed over the chamfer distance between
// both: multiplied by decay per orthogonal step and decay^sqrt(2) per
// diagonal step, with decay = exp(-CellSize*ln(2) / halfDistance), just
// like PropagateAttraction.
//
// Propagation works on closed octants, each spanned by an axis step and a
// diagonal step: e.g. the closed octant 0 <= dy <= dx by (1, 0) and (1, 1).
// Every cell of the closed octant is reached from its tip by some sequence
// of these two steps, and no other cell is, and the shortest chamfer path
// to an octant cell only uses these two steps. Propagating seeds backwards
// along the two steps, with max-relaxation, is thus exact: a cell's closed
// value V (which includes the cell's own seed) is the max over its own seed
// and the values V of its two step neighbours, each multiplied by its step's
// decay. Since the decay per step is a constant factor, this never drops a
// source that would win farther on, i.e. a nearer but weaker source never
// shadows a farther but stronger one.
//
// The value published for a half-open octant uses only one of the two step
// neighbours: an octant without its diagonal ray (e.g. octant 0,
// 0 <= dy < dx) is exactly the closed octant of the neighbour one axis step
// away, shifted by that step, since every cell not on the diagonal ray has
// a shortest path starting with an axis step. Likewise, an octant without
// its axis ray (e.g. octant 1, 0 < dx <= dy) is exactly the closed octant of
// the neighbour one diagonal step away. With the starting ray included in
// each octant, even octants exclude their diagonal ray and publish the axis
// step neighbour's value, odd octants exclude their axis ray and publish
// the diagonal step neighbour's value. This also excludes the cell itself
// as its own source.
//
// The two octants of each quadrant are computed in one sweep, so there are
// four sweeps. Both steps of an octant advance by one column (the axis step
// of a y-major octant by one row instead), so each sweep only needs the
// previous and the current column of propagated values, kept in buf instead
// of full grids.
//
// On ties in the propagated values V, the cell's own seed wins over a step
// neighbour's value, and the axis step wins over the diagonal step.
//
// Mirrors the sibling Julia implementation's propagate_octants!.
func PropagateOctants(sources *res.Grid[res.OctantSources], seeds *res.Grid[float64], halfDistance float64, buf *OctantBuffers) {
	decay := math.Exp(-float64(seeds.CellSize()) * math.Ln2 / halfDistance)
	diagonalDecay := math.Pow(decay, math.Sqrt2)

	// Quadrant (sx, sy) contains octants with offsets of signs (sx, sy), and
	// its x-major octant (|dy| <= |dx|) and y-major octant (|dx| <= |dy|)
	// are kx and ky.
	propagateQuadrant(sources, seeds, buf, 1, 1, 0, 1, decay, diagonalDecay)
	propagateQuadrant(sources, seeds, buf, -1, 1, 3, 2, decay, diagonalDecay)
	propagateQuadrant(sources, seeds, buf, -1, -1, 4, 5, decay, diagonalDecay)
	propagateQuadrant(sources, seeds, buf, 1, -1, 7, 6, decay, diagonalDecay)
}

// propagateQuadrant performs the sweep of PropagateOctants for the octants
// kx (x-major) and ky (y-major) of quadrant (sx, sy).
//
// A cell's sources lie in direction (sx, sy), so the sweep starts at the
// opposite corner of the grid and processes columns, and rows within a
// column, against that direction: both steps of the x-major octant, (sx, 0)
// and (sx, sy), lead into the previous column; the y-major octant's axis
// step (0, sy) leads to the previous row of the current column, its
// diagonal step (sx, sy) into the previous column.
func propagateQuadrant(
	sources *res.Grid[res.OctantSources], seeds *res.Grid[float64], buf *OctantBuffers,
	sx, sy, kx, ky int, decay, diagonalDecay float64,
) {
	w, h := seeds.Width(), seeds.Height()

	xStep := [2]int16{int16(sx), 0}
	yStep := [2]int16{0, int16(sy)}
	diagonalStep := [2]int16{int16(sx), int16(sy)}

	// Even octants publish their axis step candidate, odd octants their
	// diagonal step candidate (see PropagateOctants).
	kxAxis := kx%2 == 0
	kyAxis := ky%2 == 0

	prevXValues, curXValues, prevYValues, curYValues := buf.values[0], buf.values[1], buf.values[2], buf.values[3]
	prevXOffsets, curXOffsets, prevYOffsets, curYOffsets := buf.offsets[0], buf.offsets[1], buf.offsets[2], buf.offsets[3]
	for i := range 4 {
		clear(buf.values[i])
		clear(buf.offsets[i])
	}

	for ix := range w {
		x := ix
		if sx > 0 {
			x = w - 1 - ix
		}
		for iy := range h {
			y := iy
			if sy > 0 {
				y = h - 1 - iy
			}
			i := y + 1      // buffer index of row y
			j := y + sy + 1 // buffer index of the row a y-step leads to
			seed := seeds.Get(x, y)

			axV, axO := octantCandidate(prevXValues[i]*decay, prevXOffsets[i], xStep)
			dxV, dxO := octantCandidate(prevXValues[j]*diagonalDecay, prevXOffsets[j], diagonalStep)
			curXValues[i], curXOffsets[i] = octantPropagated(seed, axV, axO, dxV, dxO)
			vx, ox := dxV, dxO
			if kxAxis {
				vx, ox = axV, axO
			}

			ayV, ayO := octantCandidate(curYValues[j]*decay, curYOffsets[j], yStep)
			dyV, dyO := octantCandidate(prevYValues[j]*diagonalDecay, prevYOffsets[j], diagonalStep)
			curYValues[i], curYOffsets[i] = octantPropagated(seed, ayV, ayO, dyV, dyO)
			vy, oy := dyV, dyO
			if kyAxis {
				vy, oy = ayV, ayO
			}

			r := sources.Get(x, y)
			r.Offsets[kx], r.Attraction[kx] = ox, float32(vx)
			r.Offsets[ky], r.Attraction[ky] = oy, float32(vy)
			sources.Set(x, y, r)
		}
		prevXValues, curXValues = curXValues, prevXValues
		prevXOffsets, curXOffsets = curXOffsets, prevXOffsets
		prevYValues, curYValues = curYValues, prevYValues
		prevYOffsets, curYOffsets = curYOffsets, prevYOffsets
	}
}

// octantCandidate returns the value v propagated to the current cell via a
// step s from a step neighbour, with its source offset relative to the
// current cell, i.e. the neighbour's offset o plus the step. Returns value
// 0 and offset (0, 0) if v is not positive, i.e. if there is no source.
func octantCandidate(v float64, o, s [2]int16) (float64, [2]int16) {
	if v > 0 {
		return v, [2]int16{o[0] + s[0], o[1] + s[1]}
	}
	return 0, [2]int16{}
}

// octantPropagated returns the closed value V of the current cell, and its
// source offset: the max over the cell's own seed (offset (0, 0)) and the
// axis and diagonal step candidates (av, ao) and (dv, do). On ties, the
// seed wins over the steps, and the axis step over the diagonal step.
func octantPropagated(seed, av float64, ao [2]int16, dv float64, do [2]int16) (float64, [2]int16) {
	bestV, bestO := dv, do
	if av >= dv {
		bestV, bestO = av, ao
	}
	if seed >= bestV {
		return seed, [2]int16{}
	}
	return bestV, bestO
}
