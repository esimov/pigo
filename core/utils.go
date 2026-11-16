package pigo

// abs returns the absolute value of the provided number
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// pow is a fast multiply operator meant to replace the built-in math.Pow function
// for better performance, where the speed is much important than correctness.
func pow(base float64, exp int) float64 {
	result := 1.0
	for exp > 0 {
		if exp%2 == 1 {
			result *= base
		}
		exp >>= 1
		base *= base
	}
	return result
}
