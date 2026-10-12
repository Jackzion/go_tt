// Package memo 是书里 9.7 节的 memo1 版本：
// 一个没有任何并发保护的缓存 —— 留给竞争检查器当靶子。
package memo

import "sync"

// Func 是要被缓存的昂贵函数类型（对应书里的 httpGetBody）。
type Func func(key string) (any, error)

// Result 是一次函数调用的结果（值 + 错误），缓存的基本单元。
type Result struct {
	value any
	err   error
}

// Memo 缓存：key -> Result。
// 注意：没有 mutex！map 也不是并发安全的 —— 这就是问题所在。
type Memo struct {
	f     Func
	mu    sync.Mutex
	cache map[string]Result
}

// New 创建一个 Memo。
func New(f Func) *Memo {
	return &Memo{f: f, cache: make(map[string]Result)}
}

// Get 查缓存：命中直接返回；未命中则调用 f 并存起来。
func (memo *Memo) Get(key string) (any, error) {
	memo.mu.Lock()
	res, ok := memo.cache[key] // ① 读 map
	if !ok {
		value, err := memo.f(key) // ② 没命中，做昂贵操作
		res = Result{value, err}
		memo.cache[key] = res // ③ 写 map
	}
	memo.mu.Unlock()

	return res.value, res.err
}
