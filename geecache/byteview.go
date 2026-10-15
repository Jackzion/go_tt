// Package geecache 提供了一个带 LRU 缓存策略的并发缓存系统。
//
// 核心组件包括：
//   - ByteView：不可变的字节数组视图，用于存储缓存值
//   - lru.Cache：基于 LRU 策略的缓存淘汰机制
package geecache

// ByteView 是一个不可变的字节数组视图
type ByteView struct {
	b []byte
}

// String 返回字节数组的字符串表示
func (v ByteView) String() string {
	return string(v.b)
}

// Len 返回字节数组的长度
func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice 返回字节数组的克隆
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// cloneBytes 克隆字节数组
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
