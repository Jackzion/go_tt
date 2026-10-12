// Package memo 是书里 9.7 节的 memo5：monitor 模式的缓存（本章完全体）。
//
// 设计要点：
//   - cache 是 server goroutine 的局部变量 —— 天生并发安全，一把锁都不用
//   - Memo 结构体里只有一条 requests channel（"信箱"）
//   - server 永不阻塞：重活派给 call 工人，送信派给 deliver 跑腿
//   - 重复抑制沿用 entry.ready 广播（和 memo4 相同）
//   - Close() 关闭信箱，server 的 for-range 结束，goroutine 优雅退出
package memo

// Func 是要被缓存的昂贵函数类型。
type Func func(key string) (any, error)

// result 是一次函数调用的结果。
type result struct {
	value any
	err   error
}

// request 是发给 server 的消息：要算哪个 key + 客户的回信专线。
type request struct {
	key      string
	response chan<- result // 客户只想要一个结果
}

// entry 是缓存条目（和 memo4 相同）：ready 关闭即广播"res 已就绪"。
type entry struct {
	res   result
	ready chan struct{}
}

// call 是干活的工人：求值 f，然后广播。
func (e *entry) call(f Func, key string) {
	e.res.value, e.res.err = f(key)
	close(e.ready) // 广播就绪
}

// deliver 是跑腿：等广播，然后把结果发给某一位客户的私人专线。
func (e *entry) deliver(response chan<- result) {
	<-e.ready        // 等结果（可能等很久，没关系，server 不等我）
	response <- e.res // 发给这一单的客户
}

// Memo 是并发安全的缓存。用完必须调 Close() 关停 server。
type Memo struct {
	requests chan request
}

// New 返回 f 的记忆化版本，并启动 server goroutine。
func New(f Func) *Memo {
	memo := &Memo{requests: make(chan request)}
	go memo.server(f)
	return memo
}

// Get 查缓存：投信 → 等回信 → 拆信。
func (memo *Memo) Get(key string) (any, error) {
	response := make(chan result)           // 每单一条私人回执专线
	memo.requests <- request{key, response} // 投信
	res := <-response                       // 等回信
	return res.value, res.err
}

// Close 关闭信箱，server 处理完队列中的信后退出。
// 警告：Close 之后再调 Get 会 panic（向已关闭的 channel 发送）。
func (memo *Memo) Close() { close(memo.requests) }

// server 是 monitor goroutine：cache 被"封闭"在它肚子里，别人绝摸不到。
func (memo *Memo) server(f Func) {
	cache := make(map[string]*entry) // ★ 全程无锁的秘密：map 是我的私有局部变量
	for req := range memo.requests { // 循环读信，直到 Close 关门
		e := cache[req.key]
		if e == nil {
			// 第一次请求这个 key：建占位符，派工人去算。
			e = &entry{ready: make(chan struct{})}
			cache[req.key] = e
			go e.call(f, req.key) // 重活外包，server 继续读下一封信
		}
		// 无论新旧条目，都派一个跑腿去给这位客户送结果。
		go e.deliver(req.response)
	}
}
