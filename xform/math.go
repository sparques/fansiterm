package xform

import (
	"image/color"
	"math"

	"golang.org/x/exp/constraints"
)

type Number interface {
	constraints.Integer | constraints.Float
}

func abs[N Number](n N) N {
	if n < 0 {
		return -n
	}
	return n
}

// rot rotates a point (x,y) around (0,0), by theta radians
func rot(x, y int, theta float64) (int, int) {
	newTheta := math.Atan2(float64(y), float64(x)) + theta
	r := math.Sqrt(math.Pow(float64(y), 2) + math.Pow(float64(x), 2))
	return int(math.Round(r * math.Cos(newTheta))), int(math.Round(r * math.Sin(newTheta)))
}

func weightedAvgColor(a, b color.Color, aWeight float64) color.RGBA {
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()

	return color.RGBA{
		uint8(math.Round(float64(r1)*aWeight + float64(r2)*(1-aWeight))),
		uint8(math.Round(float64(g1)*aWeight + float64(g2)*(1-aWeight))),
		uint8(math.Round(float64(b1)*aWeight + float64(b2)*(1-aWeight))),
		uint8(math.Round(float64(a1)*aWeight + float64(a2)*(1-aWeight))),
	}
}
