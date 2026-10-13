// 练习 9.5：ping-pong 极限速度测试。
//
// 两个 goroutine 通过两个无缓冲 channel 反复打球，
// 测量每秒能完成多少个来回（以及换算成每秒多少条消息）。
//
// 用法：go run ./ch9/ex95_pingpong
package main

import (
	"fmt"
	"time"
)

const rounds = 1_000_000 // 每轮打 100 万个来回

// rally 打一轮：main 发球 n 次，goroutine 全部回球。
// 返回总耗时。一个来回 = 一条 ping + 一条 pong = 2 条消息。
func rally() time.Duration {
	ping := make(chan struct{})
	pong := make(chan struct{})

	go func() { // 对面的陪练：接球就回
		for i := 0; i < rounds; i++ {
			<-ping
			pong <- struct{}{}
		}
	}()

	start := time.Now()
	for i := 0; i < rounds; i++ {
		ping <- struct{}{} // 发球（无缓冲：陪练接住才算完成）
		<-pong             // 等回球
	}
	return time.Since(start)
}

func main() {
	fmt.Printf("每轮 %d 万个来回（= %d 万条消息），共 5 轮：\n\n", rounds/10000, rounds*2/10000)

	var totalRPS float64
	for trial := 1; trial <= 5; trial++ {
		d := rally()
		rps := float64(rounds) / d.Seconds()        // 往返次/秒
		msgs := float64(rounds*2) / d.Seconds()     // 消息条/秒
		perRallyNs := float64(d.Nanoseconds()) / float64(rounds) // 每来回纳秒
		totalRPS += rps
		fmt.Printf("第%d轮: %12v | 往返 %9.0f 次/s | 消息 %9.0f 条/s | 每来回 %5.0f ns（每条消息 %.0f ns）\n",
			trial, d, rps, msgs, perRallyNs, perRallyNs/2)
	}
	fmt.Printf("\n平均: %.0f 个来回/s（%.0f 条消息/s）\n", totalRPS/5, totalRPS*2/5)
}
