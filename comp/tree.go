package comp

// Position component with integer cell coordinates.
type Position struct {
	X int
	Y int
}

// Damaged marks a tree as damaged in the perception of the vector.
type Damaged struct{}

// Infected marks a tree as infested with PWN.
type Infected struct {
	InfectionTick int
}

// Colonized marks a tree as colonized by beetles (eggs, larvae or pupae).
type Colonized struct{}
