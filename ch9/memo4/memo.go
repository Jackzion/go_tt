// Package memo 是书里 9.7 节的最终版 memo4：duplicate suppression（重复抑制）。
//
// 设计要点：
//   - cache 里存 *entry（占位符），entry 在结果算出来之前就进 map
//   - 谁创建 entry，谁负责算 f 并 close(ready) 广播
//   - 后来者发现 entry 已存在，就 <-e.ready 等广播，绝不重复计算
//   - 锁只保护 map 的读写，昂贵的 f 在锁外执行 → 不同 key 完全并行
package memo

import "sync"

// Func 是要被缓存的昂贵函数类型。
type Func func(key string) (any, error)

// result 是一次函数调用的结果。
type result struct {
	value any
	err   error
}

// entry 是缓存表里的一个条目：
// res 存结果；ready 是"结果已就绪"的广播信号（关闭即广播）。
type entry struct {
	res   result
	ready chan struct{} // closed when res is ready
}

// Memo 是并发安全的缓存。
type Memo struct {
	f     Func
	mu    sync.Mutex // 只保护 cache 这张表，不保护计算过程
	cache map[string]*entry
}

// New 创建一个 Memo。
func New(f Func) *Memo {
	return &Memo{f: f, cache: make(map[string]*entry)}
}

// Get 查缓存。同 key 并发请求只会触发一次 f，其余人等广播。
func (memo *Memo) Get(key string) (value any, err error) {
	memo.mu.Lock()
	e := memo.cache[key]
	if e == nil {
		// 第一次请求这个 key：本 goroutine 负责计算并广播"就绪"。
		e = &entry{ready: make(chan struct{})}
		memo.cache[key] = e // 占位符先进 map，让后来者能看到"有人在办"
		memo.mu.Unlock()    // 放锁！f 很贵，不能占着锁算

		e.res.value, e.res.err = memo.f(key) // 锁外执行昂贵操作

		close(e.ready) // 广播：所有等这个 key 的人，醒来拿结果！
	} else {
		// 重复请求：entry 已在表里，等着广播就行。
		memo.mu.Unlock()
		<-e.ready // 阻塞直到 close(e.ready)
	}
	return e.res.value, e.res.err
}
