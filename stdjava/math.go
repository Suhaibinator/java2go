package stdjava

import (
	"math"
)

// This file implements the java.lang.Math operations whose Go equivalents
// either are not type-preserving (math.Abs/Max/Min are float64-only) or have a
// different return contract (Math.round returns long via round-half-up).

// The `number` constraint used by these helpers is declared in common.go.

// MathAbs returns the absolute value of x, preserving its numeric type, matching
// Java's overloaded Math.abs (which returns the same type it is given rather than
// always a float64).
func MathAbs[T number](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

// MathMax returns the larger of a and b, preserving the numeric type, matching
// Java's overloaded Math.max.
func MathMax[T number](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// MathMin returns the smaller of a and b, preserving the numeric type, matching
// Java's overloaded Math.min.
func MathMin[T number](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// MathRound rounds x to the nearest long using round-half-up (ties round toward
// positive infinity), matching Java's Math.round(double). Go's math.Round rounds
// half away from zero, which differs for negative .5 values, so this is computed
// explicitly.
func MathRound(x float64) int64 {
	return int64(math.Floor(x + 0.5))
}
