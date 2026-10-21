// Package lru 实现了一个基于 LRU（Least Recently Used，最近最少使用）策略的内存缓存。
//
// 数据结构设计：
//
//	双向链表（container/list）：维护元素的"新鲜度"顺序
//	    - 链表头部（front）= 最近被访问的元素
//	    - 链表尾部（back） = 最久未被访问的元素（首选淘汰对象）
//	哈希表 map[string]*list.Element：key -> 链表节点的映射，
//	    使得查找、更新都能在 O(1) 时间内完成
//
// 核心流程：
//
//	Add          写入或更新缓存；若总字节数超过 maxBytes，循环淘汰链表尾部节点
//	Get          读取缓存；命中后把节点移动到链表头部（标记为"最近使用"）
//	RemoveOldest 删除链表尾部节点、扣减字节数，必要时触发 OnEvicted 回调
//
// 注意：本包自身不加锁，非并发安全；并发安全由上层 geecache.cache 通过互斥锁保证。
package lru

import "container/list"

// Cache 代表一个 LRU 缓存实例。
// 请通过 New 函数创建，不要直接构造 Cache 结构体（否则内部字段未初始化，无法使用）。
type Cache struct {
	maxBytes  int64                         // 缓存容量上限（字节）；为 0 表示不限制
	nbytes    int64                         // 当前已占用字节数（所有 key + value 的字节总和）
	ll        *list.List                    // 双向链表：保存缓存元素，并按最近使用程度排序
	cache     map[string]*list.Element      // 哈希表：key -> 链表节点，用于 O(1) 查找
	OnEvicted func(key string, value Value) // 淘汰回调：元素被淘汰时触发，可用于释放资源或做统计
}

// entry 是链表节点中实际存储的键值对。
// key 需要单独保存，是因为链表节点只能存一个 value，
// 而淘汰/更新时都必须知道对应的 key。
type entry struct {
	key   string
	value Value
}

// Value 是缓存值的抽象接口。
// 只需实现 Len() int 告诉缓存该值占用多少字节，
// Cache 就能计算总字节数并在超限时触发淘汰。
// （geecache 包中的 ByteView 实现了该接口）
type Value interface {
	Len() int
}

// New 创建一个 LRU 缓存。
//
// 参数：
//
//	maxBytes  缓存容量上限（字节）；传 0 表示不限制容量
//	onEvicted 元素被淘汰时的回调函数，可传 nil 表示不需要回调
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		nbytes:    0,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get 根据 key 查找缓存值。
//
// 命中时：
//  1. 将对应节点移动到链表头部（表示最近使用过，不会被优先淘汰）
//  2. 返回 (value, true)
//
// 未命中时：
//  3. 返回 (nil, false)
func (c *Cache) Get(key string) (value Value, ok bool) {
	if e, ok := c.cache[key]; ok {
		c.ll.MoveToFront(e)
		value = e.Value.(*entry).value
		return value, true
	}
	return nil, false
}

// RemoveOldest 删除链表尾部的元素（即最久未被使用的元素）。
//
// 步骤：
//  1. 取链表尾部节点
//  2. 从链表和哈希表中移除
//  3. 扣减已占用字节数
//  4. 若设置了 OnEvicted 回调，则触发回调
func (c *Cache) RemoveOldest() {
	// 取链表尾部节点
	ele := c.ll.Back()
	if ele != nil {
		// 从链表中移除
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		// 从哈希表中删除
		delete(c.cache, kv.key)
		// 字节数扣减规则必须与 Add 中的累加规则保持一致：
		// 新增项累加 len(key) + value.Len()，故淘汰时按同样口径扣减
		c.nbytes -= int64(kv.value.Len()) + int64(len(kv.key))
		if c.OnEvicted != nil {
			// 调用淘汰回调
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add 写入或更新一个缓存项。
//
// 逻辑：
//   - key 已存在：更新 value，移动到链表头部，只调整新旧 value 的字节数差值
//   - key 不存在：新建节点插入链表头部，并累加 key 和 value 的字节数
//   - 写入后若总字节数超过 maxBytes，则循环淘汰最旧的元素，直到不再超限
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		// key 已存在：移动到链表头部，标记为最近使用
		c.ll.MoveToFront(ele)
		// 更新 value，并按差值调整已占用字节数
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len() - kv.value.Len())
		kv.value = value
	} else {
		// key 不存在：插入链表头部，并登记到哈希表
		c.ll.PushFront(&entry{key: key, value: value})
		c.cache[key] = c.ll.Front()
		c.nbytes += int64(value.Len()) + int64(len(key))
	}
	// 超限则持续淘汰最旧元素（maxBytes 为 0 表示不限量，永不淘汰）
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Len 返回当前缓存中的元素个数。
func (c *Cache) Len() int {
	return c.ll.Len()
}
