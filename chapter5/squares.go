package chapter5

// 匿名函数。每次返回下一个数的平方
func Squares() func() int {
	var x int
	return func() int {
		x++
		return x * x
	}
}
