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

// WalkDirection component of beetles.
//
// Movement state that a beetle keeps between movement steps (see
// sys.BeetleMovement) while it is on the way, i.e. when it ran out of
// distance budget before arriving:
//
//   - Dx, Dy: the direction of its walk away from a tree it left, if the
//     walk ended before the beetle left the area attracted by that tree. A
//     unit vector, or (0, 0) if the beetle is not walking away from a tree.
//     The beetle continues the walk in the next step.
//   - Feeding: the beetle's mode (feeding, or egg laying), kept in the next
//     step instead of drawing it anew.
//   - OnTheWay: whether the beetle is walking away from a tree, or flying
//     towards a target tree it has not reached yet.
//
// All beetles have this component. The zero value is the state of a beetle
// that is not on the way.
//
// The direction is stored as float32 to save memory, and converted back to
// float64 for all calculations. Both conversions are exact or round to
// nearest, the same in the sibling Julia implementation.
type WalkDirection struct {
	Dx       float32
	Dy       float32
	Feeding  bool
	OnTheWay bool
}
