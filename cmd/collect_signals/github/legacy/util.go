package legacy

import (
	"math"
	"time"
)

func TimeDelta(a, b time.Time, u time.Duration) int {
	var d time.Duration
	if a.Before(b) {
		d = b.Sub(a)
	} else {
		d = a.Sub(b)
	}
	return int(d / u)
}

// Round will return v approximately rounded to a precision of p decimal places.
func Round(v float64, p int) float64 {
	m := math.Pow10(p)
	return math.Round(v*m) / m
}
