package app

import "math"

func strictBoundedInt(value interface{}, low, high int) (int, bool) {
	switch v := value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Trunc(v) != v || v < float64(low) || v > float64(high) {
			return 0, false
		}
		return int(v), true
	case int:
		return v, v >= low && v <= high
	default:
		return 0, false
	}
}
