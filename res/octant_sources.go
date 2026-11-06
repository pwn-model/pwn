package res

// OctantSources holds, for one cell, the most attractive source tree in each
// of the 8 octants around that cell.
//
// Octant k (0-based) covers the directions from 45°*k (included) to
// 45°*(k+1) (excluded), measured from +x towards +y (half-open octants), so
// the octant of a direction at angle a is floor(a / 45°). With (dx, dy) the
// offset from the cell to a source:
//
//	0: 0 <=  dy <   dx      4: 0 <= -dy <  -dx
//	1: 0 <   dx <=  dy      5: 0 <  -dx <= -dy
//	2: 0 <= -dx <   dy      6: 0 <=  dx <  -dy
//	3: 0 <   dy <= -dx      7: 0 <  -dy <=  dx
//
// Every cell other than the cell itself is in exactly one octant. The cell
// itself is in none: a tree's own cell is excluded, so that a beetle on a
// tree "sees" only other trees.
//
// The layout is 64 bytes, i.e. one cache line, so that a consumer selecting
// among a cell's octants reads them all with a single memory access.
//
// Octant k here is octant k+1 of the sibling Julia implementation, whose
// octants are 1-based.
type OctantSources struct {
	// Offsets are the (dx, dy) offsets, in cells, from the cell to the source
	// of each octant. (0, 0) for an octant without any source: since the
	// cell itself is never a source, (0, 0) is never a valid offset.
	Offsets [8][2]int16
	// Attraction is the attraction value of each octant's source as seen
	// from the cell, i.e. its seed decayed over the distance to the cell. 0
	// for an octant without any source.
	Attraction [8]float32
}
