package geometry

import (
	"fmt"
	"image/color"
)

// ColoredPoint is a Point with an associated color.
type ColoredPoint struct {
	*Point
	Color color.RGBA
}

func Test() {

	red := color.RGBA{255, 0, 0, 255}
	blueL := color.RGBA{0, 0, 255, 255}
	var p = ColoredPoint{&Point{1, 2}, red}
	var q = ColoredPoint{&Point{5, 6}, blueL}
	p.Point = q.Point
	fmt.Println(p.Distance(*q.Point))

}