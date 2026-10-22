package singleflight

import "sync"

type call struct {
	wg  sync.WaitGroup // 用于等待所有调用完成
	val interface{}    // 存储调用结果
	err error          // 存储调用错误信息
}

// Group 是 singleflight 的主数据结构，管理不同 key 的请求(call)。
// 每个键对应一个 call，用于记录该键的调用情况。
type Group struct {
	mu sync.Mutex
	m  map[string]*call // key -> call
	// 每个 call 都对应一个 wg，用于等待所有调用完成。
}

// Do 方法，接收 2 个参数，第一个参数是 key，第二个参数是一个函数 fn。
// Do 的作用就是，针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误。
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 如果请求正在进行中，等待请求完成 然后返回结果。
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	// 发起新的请求
	c := new(call)
	// 发起请求前加锁
	c.wg.Add(1)
	// 添加到 g.m 表面 key 已经有对应的请求在处理
	g.m[key] = c
	g.mu.Unlock()
	// 调用 fn 发起请求 , 并赋值给 call
	c.val, c.err = fn()
	// 请求结束，释放锁
	c.wg.Done()

	// 更新 g.m
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	return c.val, c.err
}
