package jmath

import "math"

// Round1 returns the rounded value of x to specified precision, use ROUND_HALF_UP mode.
// For example:
//
//		  Round1(0.6635, 3)   // 0.664
//	   Round1(0.363636, 2) // 0.36
//	   Round1(0.363636, 1) // 0.4
func RoundWithPrecision(x float64, precision int) float64 {
	if precision == 0 {
		return math.Round(x)
	}

	p := math.Pow10(precision)
	if precision < 0 {
		return math.Round(x*p) * math.Pow10(-precision)
	}
	return math.Round(x*p) / p
}
