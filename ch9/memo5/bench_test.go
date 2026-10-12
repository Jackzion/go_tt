// memo4 vs memo5 基准对比 —— 用数据回答"memo5 一定更快吗"。
//
//	go test -run '^$' -bench Hit   -benchtime=1s ./ch9/memo5   缓存命中开销
//	go test -run '^$' -bench Dedup -benchtime=500ms ./ch9/memo5  100 人抢一个 key
package memo

import (
	"fmt"
	"sync"
	"testing"
	"time"

	memo4 "go_tt/ch9/memo4" // 别名：memo4 包的本名也叫 memo，必须显式区分
)

// api 是两个版本共同的最小接口，方便同一段测试代码驱动它们。
type api interface {
	Get(key string) (any, error)
}

// cheapF 几乎免费的 f —— 用来放大"每次 Get 的固定开销"。
func cheapF(key string) (any, error) { return []byte("data:" + key), nil }

// slowF 模拟 1ms 的网络请求 —— 用来观察"f 很贵时两者是否打平"。
func slowF(key string) (any, error) {
	time.Sleep(time.Millisecond)
	return []byte("data:" + key), nil
}

var benchKeys = func() []string {
	keys := make([]string, 64)
	for i := range keys {
		keys[i] = fmt.Sprintf("https://example.com/page/%d", i)
	}
	return keys
}()

// benchmarkHit：64 个 key 全部预热（缓存命中场景），然后多 goroutine 并发读。
func benchmarkHit(b *testing.B, m api) {
	for _, k := range benchKeys {
		if _, err := m.Get(k); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if _, err := m.Get(benchKeys[i%len(benchKeys)]); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkMemo4_Hit(b *testing.B) {
	m := memo4.New(cheapF)
	benchmarkHit(b, m)
}

func BenchmarkMemo5_Hit(b *testing.B) {
	m := New(cheapF)
	defer m.Close()
	benchmarkHit(b, m)
}

// benchmarkDedup：每轮新建一个缓存，100 个 goroutine 同时抢同一个 key。
// f 花 1ms —— 理想耗时约等于 1ms（只算一次），两者应该打平。
func benchmarkDedup(b *testing.B, newMemo func() (api, func())) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m, closeFn := newMemo()
		b.StartTimer()

		var wg sync.WaitGroup
		for j := 0; j < 100; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := m.Get("shared-key"); err != nil {
					b.Error(err)
				}
			}()
		}
		wg.Wait()

		b.StopTimer()
		closeFn()
		b.StartTimer()
	}
}

func BenchmarkMemo4_Dedup100(b *testing.B) {
	benchmarkDedup(b, func() (api, func()) {
		return memo4.New(slowF), func() {}
	})
}

func BenchmarkMemo5_Dedup100(b *testing.B) {
	benchmarkDedup(b, func() (api, func()) {
		m := New(slowF)
		return m, m.Close
	})
}
