package ch8

import (
	"fmt"
	"os"
	"time"
)

func TestCountdown() {
	fmt.Println("Commencing countdown.")
	tick := time.Tick(1 * time.Second)

	// abort countdown
	abort := make(chan struct{})
	go func() {
		os.Stdin.Read(make([]byte, 1))
		abort <- struct{}{}
		fmt.Println("Countdown aborted.")
	}()

	for countdown := 10; countdown > 0; countdown-- {
		fmt.Println(countdown)
		// 多路复用 ， 监听 ticker 与 abort chan
		select {
		case <-tick:
		case <-abort:
			fmt.Println("Countdown aborted.")
			return
		}
	}
	launch()
}

func launch() {
	fmt.Println("Launch!")
}
