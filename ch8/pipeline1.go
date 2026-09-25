package ch8

import (
	"fmt"
)

func Counter(out chan<- int) {
	for i := 0; i < 10; i++ {
		out <- i
	}
	close(out)
}

func Squarer(in <-chan int, out chan<- int) {
	for n := range in {
		out <- n * n
	}
	close(out)
}

func Printer(in <-chan int) {
	for n := range in {
		fmt.Println(n)
	}
}

func TestPipeline1() {

	naturals := make(chan int)
	squares := make(chan int)
	go Counter(naturals)
	go Squarer(naturals, squares)
	Printer(squares)
}
