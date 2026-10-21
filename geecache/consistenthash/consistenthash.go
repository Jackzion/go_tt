// Package consistenthash 实现了一致性哈希算法。
//
// 核心思想：
//   - 将所有节点（真实节点 + 虚拟节点）映射到一个 0 ~ 2^32-1 的哈希环上
//   - 每个真实节点对应 replicas 个虚拟节点，使节点在环上分布更均匀，
//     避免真实节点较少时数据严重倾斜
//   - 查找 key 时，计算 key 的哈希值，顺时针方向找到的第一个节点即为该 key 的归属节点
//
// 优势：增删节点时只影响相邻节点的数据，最大程度减少数据迁移量。
package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash 是自定义哈希函数的类型，便于替换为更合适的哈希算法（如 murmurhash）。
type Hash func(data []byte) uint32

// Map 是一致性哈希的核心数据结构（哈希环）。
type Map struct {
	hash     Hash           // 哈希函数，默认为 CRC32
	replicas int            // 每个真实节点对应的虚拟节点数量
	keys     []int          // 所有虚拟节点的哈希值，升序排序，便于二分查找
	hashMap  map[int]string // 虚拟节点哈希值 -> 真实节点地址
}

// New 创建一个一致性哈希环。
//
// 参数：
//
//	replicas 每个真实节点对应的虚拟节点数量（越大分布越均匀，但占用内存越多）
//	fn       自定义哈希函数；传 nil 则使用默认的 CRC32
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	// 如果没有指定哈希函数，默认使用 CRC32
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 将若干真实节点加入哈希环。
// 每个真实节点会生成 replicas 个虚拟节点（节点地址前拼上序号后取哈希，
// 如 "0http://localhost:8001"、"1http://localhost:8001" ...），
// 全部加入后对 keys 重新排序，保证后续二分查找可用。
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		// 为每个真实节点创建多个虚拟节点
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Get 根据 key 找到其归属的真实节点地址。
//
// 步骤：
//  1. 计算 key 的哈希值
//  2. 二分查找第一个 >= 该哈希值的虚拟节点（即顺时针方向遇到的第一个节点）
//  3. 若超出环的末尾（idx == len(keys)），通过取模绕回环首，体现"环形"语义
//  4. 返回该虚拟节点对应的真实节点地址；环为空时返回 ""
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	// 二分查找第一个 >= hash 的虚拟节点下标
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	// 取模实现环形回绕：idx 超出末尾时回到环首
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
