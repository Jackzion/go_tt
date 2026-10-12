// 练习 9.1：在 gopl.io/ch9/bank1 的基础上添加 Withdraw 取款函数。
//
// bank1 的核心思想：变量 balance 只被 teller 这一个 goroutine 持有，
// 其他 goroutine 一律通过 channel 与它通信，因此天生并发安全，无需互斥锁。
package ch9

import "fmt"

// deposits：发送取款/存款金额给 teller
var deposits = make(chan int)

// balances：teller 通过它把余额发给查询者
var balances = make(chan int)

// withdrawals：发送取款请求给 teller。
// 请求里除了金额，还必须带一个"回执通道"，
// 因为 teller 处理完后要把成功与否的结果发回给 Withdraw。
var withdrawals = make(chan withdrawal)

// withdrawal 是一笔取款请求：
//   - amount：要取多少钱
//   - reply： 新建的 channel，teller 会把 boolean 结果发到这里
type withdrawal struct {
	amount int
	reply  chan bool
}

// Deposit 存款：把金额发给 teller
func Deposit(amount int) { deposits <- amount }

// Balance 查询余额：向 teller 要一个余额快照
func Balance() int { return <-balances }

// Withdraw 取款。
// 返回 true  表示取款成功；
// 返回 false 表示余额不足，取款失败。
func Withdraw(amount int) bool {
	reply := make(chan bool) // 每笔取款新建一个回执通道（题目要求的新 channel）
	withdrawals <- withdrawal{amount, reply}
	return <-reply // 阻塞在这里，直到 teller 把结果发回来
}

// teller 就是 monitor goroutine：它是唯一碰 balance 变量的 goroutine。
func teller() {
	var balance int // balance 被"封闭"在 teller 内部，外界永远摸不到
	for {
		select {
		case amount := <-deposits:
			balance += amount
		case w := <-withdrawals:
			if w.amount > balance {
				w.reply <- false // 余额不足：拒绝，不动 balance
				continue
			}
			balance -= w.amount // 够钱：扣款
			w.reply <- true     // 回执：成功
		case balances <- balance:
			// 有人查询余额，把当前快照发出去
		}
	}
}

// init 在包初始化时启动 teller（monitor）goroutine
func init() {
	go teller()
}

func main() {
	// ---- 演示 1：并发存款 ----
	// 10 个 goroutine 各存 10 元，互不加锁也不会出错
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			Deposit(10)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done // 等 10 笔存款全部完成
	}
	fmt.Println("存完 10x10 后余额:", Balance()) // 100

	// ---- 演示 2：取款（本练习的重点）----
	fmt.Println("取 30 :", Withdraw(30)) // true，余额还剩 70
	fmt.Println("取 80 :", Withdraw(80)) // false，余额不足！
	fmt.Println("取 70 :", Withdraw(70)) // true，把剩下的全取走
	fmt.Println("取 1 :", Withdraw(1))   // false，已经是 0 了
	fmt.Println("最终余额:", Balance())     // 0

	// ---- 演示 3：并发取款也不会乱 ----
	Deposit(50) // 先存 50
	var okCount, failCount int
	c := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			c <- Withdraw(20) // 5 个人都想取 20，但总共只有 50
		}()
	}
	for i := 0; i < 5; i++ {
		if <-c {
			okCount++
		} else {
			failCount++
		}
	}
	fmt.Printf("5 笔并发取款：成功 %d 笔，失败 %d 笔，余额 %d（应为 10）\n",
		okCount, failCount, Balance())
}
