package comp

// BeetlePosition component with integer cell coordinates.
type BeetlePosition struct {
	X int
	Y int
}

// EmergenceTick component of beetles.
type EmergenceTick struct {
	TickOfEmergence int
}

// LifeExpectancy component of beetles.
type LifeExpectancy struct {
	TickOfDeath int
}
