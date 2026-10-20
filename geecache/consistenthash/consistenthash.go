package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

// Map 是一个一致性哈希映射
type Map struct {
	hash     Hash           // 哈希函数
	replicas int            // 虚拟节点数量
	keys     []int          // 虚拟节点的哈希值，按升序排序
	hashMap  map[int]string // 虚拟节点的哈希值到真实节点的映射关系
}

// New 创建一个一致性哈希映射
// replicas 是虚拟节点数量
// fn 是哈希函数
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

// Add 添加真实节点到一致性哈希映射
// keys 是真实节点的键值对
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		// 为每个真实节点添加多个虚拟节点
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Get gets the closest item in the hash to the provided key.
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	// 二分查找的一个大于 hash 的 key_idx
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
