package geecache

// ByteView 是一个不可变（immutable）的字节数组视图，用于在缓存中保存数据。
//
// "不可变"意味着一旦写入缓存，外部就无法修改其内部字节：
//   - 读取时通过 ByteSlice 返回内部字节的克隆副本，而不是内部切片本身
//   - 因此多个 goroutine、多个缓存场景可以安全地共享同一份数据，无需加锁
//
// ByteView 同时实现了 lru.Value 接口（Len 方法），
// 所以可以直接作为值存入 lru.Cache。
type ByteView struct {
	b []byte // 缓存的真实数据（只读约定，不对外暴露）
}

// String 以字符串形式返回缓存值。
// 字符串本身不可变，转换过程即复制，天然安全；常用于日志输出或按文本使用。
func (v ByteView) String() string {
	return string(v.b)
}

// Len 返回缓存值的字节长度。
// 实现了 lru.Value 接口，供 LRU 缓存计算容量、触发淘汰使用。
func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice 返回缓存值内容的克隆副本。
// 返回副本而不是内部切片，是为了防止调用方修改数据、破坏"不可变"约定。
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// cloneBytes 复制一份字节切片，防止外部持有并修改原始数据。
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
