// 数据竞争演示：5 个 goroutine 并发给 counter 加 1，
// 没有任何同步措施 —— 教科书级的 data race。
//
// 运行方式：
//
//	go run ch9/race_demo.go          结果随机错误，但可能"看起来正常"
//	go run -race ch9/race_demo.go    竞争检查器精确抓出问题（需要 cgo）
package ch9

import (
	"fmt"
	"sync"
)

// go run -race ch9/race_demo.go
func TestRaceDemo() {
	var counter int // 共享变量，没有任何保护

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				counter++ // ← 危险！这不是原子操作：读→加1→写，三步可被打断
			}
		}()
	}
	wg.Wait()
	fmt.Println("counter =", counter, "(正确答案应该是 5000)")
}
