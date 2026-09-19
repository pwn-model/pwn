package util

// CeilDiv calculates ceiled integer division.
func CeilDiv(a, b int) int {
	return (a + b - 1) / b
}
