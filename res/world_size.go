package res

// WorldSize resource
type WorldSize struct {
	Width      int // Width of the world in tree diameters
	Height     int // Height of the world in tree diameters
	Resolution int // Resolution of the spatial grid in tree diameters per cell
}

// ToCoords calculates space grid coords from tree coords.
func (s *WorldSize) ToCoords(x, y int) (int, int) {
	return x / s.Resolution, y / s.Resolution
}
