package geecache

import (
	"go_tt/geecache/lru"
	"sync"
)

// cache 是对 lru.Cache 的并发安全封装。
//
// 之所以需要再包一层：
//  1. lru.Cache 本身不加锁，cache 通过互斥锁保证并发安全
//  2. lru.Cache 存储的是 interface{}，cache 将类型限定为 ByteView，省去上层类型断言的麻烦
//  3. 支持懒初始化：创建 Group 时先不建 lru.Cache，等第一次写入时再用 cacheBytes 创建
type cache struct {
	mu         sync.Mutex // 互斥锁：保护下方所有字段的并发访问
	lru        *lru.Cache // 底层 LRU 缓存；为 nil 表示尚未初始化
	cacheBytes int64      // 缓存容量上限（字节），透传给 lru.Cache
}

// add 写入一个缓存项。
// 若底层 lru.Cache 尚未初始化，则先用 cacheBytes 完成懒初始化。
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

// get 读取一个缓存项。
// 返回 (value, true) 表示命中；返回 (零值, false) 表示未命中或尚未初始化。
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}
	return
}
