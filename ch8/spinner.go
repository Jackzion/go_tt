// Package ch8 provides a spinner function that displays a spinning animation in the console.
package ch8

import "time"

func Spinner(delay time.Duration) {
	// Implementation for spinner function
	for {
		for _, r := range `-\|/` {
			print("\r", string(r))
			time.Sleep(delay)
		}
	}
}

func Fib(x int) int {
	if x < 2 {
		return x
	}
	return Fib(x-1) + Fib(x-2)
}
