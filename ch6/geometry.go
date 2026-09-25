// Package geometry provides basic geometric types and functions.
package geometry

import (
	"math"
)

type Point struct {
	X, Y float64
}

// Distance returns the distance between two points p and q.
func Distance(p, q Point) float64 {
	return math.Hypot(q.X-p.X, q.Y-p.Y)
}

func (p Point) Distance(q Point) float64 {
	return math.Hypot(q.X-p.X, q.Y-p.Y)
}

// A Path is a sequence of points.
type Path []Point

// Distance returns the total distance of the path.
func (path Path) Distance() float64 {
	sum := 0.0
	for i := range path {
		if i > 0{
			sum += path[i-1].Distance(path[i])
		}
	}
	return sum
}

func main() {
	p := Point{1, 2}
	q := Point{4, 6}
	perimeter := Path{p, q, {7, 8}, {9, 10}}
	distance := Distance(p, q)
	println("Distance between p and q:", distance) // function call
	println("Distance between p and q (method):", p.Distance(q)) // method call
	println("Total distance of the path:", perimeter.Distance()) // method call on Path
}