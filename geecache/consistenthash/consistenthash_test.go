package consistenthash

import (
	"fmt"
	"strconv"
	"testing"
)

// TestHashing 使用一个"恒等"哈希函数（把字符串直接解析成数字当作哈希值），
// 使得环上的刻度完全可预测，方便手工推演断言结果。
//
// New(3, identity) + Add("6","4","2") 生成的虚拟节点哈希为：
//
//	"6": 0+6=6,  16, 26
//	"4": 4,  14, 24
//	"2": 2,  12, 22
//
// 排序后 keys = [2, 4, 6, 12, 14, 16, 22, 24, 26]
func TestHashing(t *testing.T) {
	hash := New(3, func(key []byte) uint32 {
		i, _ := strconv.Atoi(string(key))
		return uint32(i)
	})

	hash.Add("6", "4", "2")

	testCases := map[string]string{
		"2":  "2", // 2 恰好命中虚拟节点 2 本身
		"11": "2", // 11 顺时针第一个 >= 它的是 12 -> "2"
		"23": "4", // 23 顺时针第一个 >= 它的是 24 -> "4"
		"27": "2", // 27 比所有虚拟节点都大，回绕到 keys[0]=2 -> "2"
	}

	for k, v := range testCases {
		if got := hash.Get(k); got != v {
			t.Errorf("Asking for %s, should have yielded %s, got %s", k, v, got)
		}
	}

	// Add("8") 之后新增虚拟节点 8, 18, 28
	// keys = [2, 4, 6, 8, 12, 14, 16, 18, 22, 24, 26, 28]
	hash.Add("8")

	// 27 顺时针撞到的新节点 28 -> "8"（其余 key 的归属不受影响）
	testCases["27"] = "8"

	for k, v := range testCases {
		if got := hash.Get(k); got != v {
			t.Errorf("After adding node 8, asking for %s, should have yielded %s, got %s", k, v, got)
		}
	}
}

// TestEmptyMap 空环上查询任何 key 都应返回空字符串，而不是 panic
func TestEmptyMap(t *testing.T) {
	hash := New(3, nil)
	if got := hash.Get("whatever"); got != "" {
		t.Errorf("empty map should return \"\", got %q", got)
	}
}

// TestWrapAround 显式覆盖"回绕"边界：
// 超过环上最大虚拟节点的 key 应绕回第一个虚拟节点
func TestWrapAround(t *testing.T) {
	hash := New(3, func(key []byte) uint32 {
		i, _ := strconv.Atoi(string(key))
		return uint32(i)
	})
	hash.Add("6", "4", "2") // keys = [2, 4, 6, 12, 14, 16, 22, 24, 26]

	// 大于 26 的所有 key 都应绕回 keys[0]=2 所属的 "2"
	for _, k := range []string{"27", "100", "99999"} {
		if got := hash.Get(k); got != "2" {
			t.Errorf("Get(%s) should wrap around to \"2\", got %q", k, got)
		}
	}

	// 恰好等于最大虚拟节点 26 的 key 应命中它所属的节点
	// 26 是 strconv.Itoa(2)+"6" 拼出来的，属于真实节点 "6"
	if got := hash.Get("26"); got != "6" {
		t.Errorf("Get(26) should hit virtual node 26 exactly (owned by \"6\"), got %q", got)
	}
}

// TestDistribution 使用真实 CRC32 哈希，验证多节点 + 多虚拟节点时
// 数据分布大致均匀（没有节点饿死或独大）
func TestDistribution(t *testing.T) {
	const (
		nodeCount = 10
		replicas  = 20
		keyCount  = 10000
	)

	hash := New(replicas, nil) // nil -> 默认 crc32.ChecksumIEEE
	nodes := make([]string, nodeCount)
	for i := range nodes {
		nodes[i] = fmt.Sprintf("node-%d", i)
	}
	hash.Add(nodes...)

	counts := make(map[string]int)
	for i := 0; i < keyCount; i++ {
		counts[hash.Get(fmt.Sprintf("key-%d", i))]++
	}

	avg := float64(keyCount) / float64(nodeCount)
	for _, node := range nodes {
		ratio := float64(counts[node]) / avg
		// replicas=20、10 节点时，经验上偏差通常在 ±30% 以内；
		// 这里放宽到 0.5~2 倍，只防止极端倾斜，保证测试稳定
		if ratio < 0.5 || ratio > 2.0 {
			t.Errorf("node %s got %d keys, avg is %.0f, ratio %.2f is too skewed",
				node, counts[node], avg, ratio)
		}
	}
}

// TestMinimalDisruption 验证一致性哈希的核心卖点：
// 新增一台机器时，只有少部分 key 需要迁移，
// 其余 key 的归属保持不变（对比 hash % N 会让 ~90% 的 key 换主人）
func TestMinimalDisruption(t *testing.T) {
	const (
		oldNodes  = 10
		replicas  = 20
		keyCount  = 2000
		maxMoved  = 0.3 // 新增 1/11 的节点，理论上 ~9% 迁移，留足冗余
	)

	hash := New(replicas, nil)
	for i := 0; i < oldNodes; i++ {
		hash.Add(fmt.Sprintf("node-%d", i))
	}

	before := make(map[string]string, keyCount)
	keys := make([]string, keyCount)
	for i := 0; i < keyCount; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		before[keys[i]] = hash.Get(keys[i])
	}

	// 新增一台机器
	hash.Add("node-new")

	moved := 0
	for _, k := range keys {
		if hash.Get(k) != before[k] {
			moved++
		}
	}

	ratio := float64(moved) / float64(keyCount)
	if ratio > maxMoved {
		t.Errorf("adding 1 node moved %.1f%% of keys (%d/%d), want < %.0f%%",
			ratio*100, moved, keyCount, maxMoved*100)
	}
}
