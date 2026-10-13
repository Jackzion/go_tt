// 练习 9.4：任意长度的流水线。
//
// 问题：① 内存耗尽前能建多少级？② 一个变量穿过整条流水线要多久？
//
// 思路：每级 = 1 个 goroutine + 1 个无缓冲 channel。
// 逐级扩大规模，实测"每级内存成本"和"每级握手延迟"，再线性外推。
//
// 用法：go run ./ch9/ex94_pipeline <阶段数>
package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"
)

// buildPipeline 构造一条 n 级流水线（全部无缓冲 channel），
// 不注入值，返回入口和出口，由调用方决定何时开始传输。
func buildPipeline(n int) (chan int, chan int) {
	in := make(chan int) // 第一条 channel c1

	out := in // out = 目前流水线的末端
	for i := 0; i < n; i++ {
		next := make(chan int)      // 新开一节"车厢"
		go func(in, out chan int) { // 两节车厢之间的转发工
			for v := range in {
				out <- v // 原样转发（每次转发 = 一次 rendezvous 握手）
			}
		}(out, next) // 入口=旧末端，出口=新开的（参数传，避开闭包陷阱）
		out = next // "末端"升级为新出口
	}
	return in, out
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "用法: pipeline <阶段数>")
		os.Exit(1)
	}
	n, err := strconv.Atoi(os.Args[1])
	if err != nil || n <= 0 {
		fmt.Fprintln(os.Stderr, "阶段数必须是正整数:", os.Args[1])
		os.Exit(1)
	}

	// 记录基线内存（排除 Go runtime 自身的固定开销）
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	// 造流水线（所有转发工此刻 park 在各自的入口上睡觉，值还没注入）
	in, out := buildPipeline(n)
	time.Sleep(200 * time.Millisecond) // 等所有 goroutine 完成调度并 park

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	totalMB := float64(m.Sys) / 1024 / 1024
	baseMB := float64(before.Sys) / 1024 / 1024
	perStageKB := float64(m.Sys-before.Sys) / float64(n) / 1024

	// 计时：现在才注入值，让它从第 1 级走到第 n 级（穿过 n+1 次无缓冲握手）
	start := time.Now()
	in <- 1 // 注入：main 阻塞到 G1 接住
	v := <-out // 等最后一级把值送到门口
	elapsed := time.Since(start)

	perHopUs := float64(elapsed.Nanoseconds()) / 1000 / float64(n+1)

	fmt.Printf("阶段数 n      : %d\n", n)
	fmt.Printf("goroutine 数  : %d\n", runtime.NumGoroutine())
	fmt.Printf("Sys 内存      : %.1f MB（基线 %.1f MB）\n", totalMB, baseMB)
	fmt.Printf("每级内存成本  : %.2f KB\n", perStageKB)
	fmt.Printf("穿过结果      : %d（正确应为 1，值被原样转发）\n", v)
	fmt.Printf("穿过总耗时    : %v\n", elapsed)
	fmt.Printf("每级握手延迟  : %.2f µs\n", perHopUs)
}
