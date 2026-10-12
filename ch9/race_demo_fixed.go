// 修复版：用互斥锁保护 counter，-race 检查通过。
package ch9

import (
	"fmt"
	"sync"
)

// go run -race ch9/race_demo_fixed.go
func TestRaceDemoFixed() {
	var mu sync.Mutex // 保护 counter 的同一把锁
	var counter int

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				mu.Lock()
				counter++ // 读→加1→写 三步被锁成一个原子操作
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	fmt.Println("counter =", counter, "(正确答案应该是 5000)")
}
