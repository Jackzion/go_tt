package chapter5

import (
	"math"
)

// Hypot returns sqrt(x*x + y*y), the length of the hypotenuse of a
// right-angled triangle with legs of length x and y.
//
// Updated by hjz on 2026-05-26: 将函数名改为大写导出，移除不可达代码
func Hypot(x, y float64) float64 {
	return math.Sqrt(x*x + y*y)
}

func Add(x, y int) int { return x + y }
func Sub(x, y int) (z int) {
	z = x - y
	return
}
func First(x int, _ int) int { return x }

func Zero(x, y int) int { return 0 }
