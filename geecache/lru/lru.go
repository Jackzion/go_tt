// Package lru 实现 LRU LRU 缓存策略
package lru

import "container/list"

type Cache struct {
	maxBytes int64                    // 最大缓存字节数
	nbytes   int64                    // 当前缓存字节数
	ll       *list.List               // LRU 缓存双向链表
	cache    map[string]*list.Element // 缓存映射表，key 是缓存键，value 是链表节点
	// OnEvicted 是缓存被逐出时的回调函数
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

// New 创建一个 LRU 缓存
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		nbytes:    0,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get 获取缓存中的值
// 如果 key 存在，返回对应的 Value 和 true , 并将该节点移动到链表头
// 如果 key 不存在，返回 nil 和 false
func (c *Cache) Get(key string) (value Value, ok bool) {
	if e, ok := c.cache[key]; ok {
		c.ll.MoveToFront(e)
		value = e.Value.(*entry).value
		return value, true
	}
	return nil, false
}

// RemoveOldest 移除缓存中的最旧项 - 链表队尾项
func (c *Cache) RemoveOldest() {
	// 取得队尾项
	ele := c.ll.Back()
	if ele != nil {
		// 移除队尾项
		c.ll.Remove(ele)
		// 从缓存映射表中删除队尾项
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		// 更新当前缓存字节数（与 Add 中的计算保持一致：value + key）
		c.nbytes -= int64(kv.value.Len()) + int64(len(kv.key))
		if c.OnEvicted != nil {
			// 调用 OnEvicted 回调函数
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add 添加或更新缓存项
// 如果 key 已存在，更新值
// 如果 key 不存在，添加新项
// 如果缓存字节数超过最大字节数，移除最旧项
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		// 移动到链表头
		c.ll.MoveToFront(ele)
		// 如果 key 已存在，更新值
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len() - kv.value.Len())
		kv.value = value
	} else {
		// 如果 key 不存在，添加新项
		c.ll.PushFront(&entry{key: key, value: value})
		c.cache[key] = c.ll.Front()
		c.nbytes += int64(value.Len()) + int64(len(key))
	}
	// 如果缓存字节数超过最大字节数，移除最旧项
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
