package app

import (
	"math"
	"testing"
)

func TestStrictBoundedIntRejectsUntrustedNumbers(t *testing.T) {
	for _, v := range []interface{}{nil, "1", 1.5, math.NaN(), math.Inf(1), float64(1 << 60), 0, -1} {
		if _, ok := strictBoundedInt(v, 1, 10000); ok {
			t.Fatalf("accepted %v", v)
		}
	}
	for _, v := range []interface{}{1, float64(10000)} {
		if _, ok := strictBoundedInt(v, 1, 10000); !ok {
			t.Fatal(v)
		}
	}
}
