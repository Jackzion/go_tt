// memo5 的系统性测试（和 memo4 的测试同款，方便对比）。
//
//	go test -v ./ch9/memo5          顺序 + 并发 + 压力都应 PASS，f 只调用 4 次
//	go test -race -v ./ch9/memo5    竞争检查器应完全安静
package memo

import (
	"sync"
	"testing"
	"time"
)

const slowCost = 50 * time.Millisecond // 模拟"昂贵操作"的耗时

// urlStream 模拟 URL 流：4 个不同 URL，每个出现两次。
func urlStream() []string {
	return []string{
		"https://golang.org",
		"https://godoc.org",
		"https://play.golang.org",
		"http://gopl.io",
		"https://golang.org",
		"https://godoc.org",
		"https://play.golang.org",
		"http://gopl.io",
	}
}

// recorder 包装昂贵函数，统计调用次数（自身带锁，不给被测对象添乱）。
type recorder struct {
	mu    sync.Mutex
	calls int
}

func (r *recorder) fn(key string) (any, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	time.Sleep(slowCost) // 模拟网络请求
	return []byte("data:" + key), nil
}

func (r *recorder) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// TestSequential：顺序使用，缓存效果和 memo1/memo4 一样好。
func TestSequential(t *testing.T) {
	r := &recorder{}
	m := New(r.fn)
	defer m.Close()

	for i, url := range urlStream() {
		start := time.Now()
		value, err := m.Get(url)
		if err != nil {
			t.Fatalf("Get(%q): %v", url, err)
		}
		elapsed := time.Since(start)
		t.Logf("%s, %s, %d bytes", url, elapsed.Truncate(time.Microsecond), len(value.([]byte)))

		if got, want := string(value.([]byte)), "data:"+url; got != want {
			t.Errorf("第%d次 Get(%q) = %q, want %q", i, url, got, want)
		}
		if i >= 4 && elapsed > 10*time.Millisecond {
			t.Errorf("第%d次 Get(%q) 花了 %v —— 应该是缓存命中", i, url, elapsed)
		}
	}
	if got := r.callCount(); got != 4 {
		t.Errorf("昂贵函数被调用 %d 次, want 4", got)
	}
}

// TestConcurrent：8 个 goroutine 并发（含重复 key）。
func TestConcurrent(t *testing.T) {
	r := &recorder{}
	m := New(r.fn)
	defer m.Close()
	urls := urlStream()

	results := make([]any, len(urls))
	errs := make([]error, len(urls))

	var n sync.WaitGroup
	for i, url := range urls {
		n.Add(1)
		go func(i int, url string) {
			defer n.Done()
			results[i], errs[i] = m.Get(url)
		}(i, url)
	}
	n.Wait()

	for i, url := range urls {
		if errs[i] != nil {
			t.Errorf("Get(%q): %v", url, errs[i])
			continue
		}
		if got, want := string(results[i].([]byte)), "data:"+url; got != want {
			t.Errorf("Get(%q) 返回 %q, want %q —— 值串台了！", url, got, want)
		}
	}
	if got := r.callCount(); got != 4 {
		t.Errorf("昂贵函数被调用 %d 次, want 4 —— 重复抑制失效", got)
	}
	t.Logf("昂贵函数实际被调用 %d 次", r.callCount())
}

// TestDedupOneKey：50 个 goroutine 抢同一个 key，f 只该被调用 1 次。
func TestDedupOneKey(t *testing.T) {
	r := &recorder{}
	m := New(r.fn)
	defer m.Close()

	var n sync.WaitGroup
	for i := 0; i < 50; i++ {
		n.Add(1)
		go func() {
			defer n.Done()
			value, err := m.Get("https://golang.org")
			if err != nil {
				t.Errorf("Get: %v", err)
				return
			}
			if got, want := string(value.([]byte)), "data:https://golang.org"; got != want {
				t.Errorf("返回 %q, want %q", got, want)
			}
		}()
	}
	n.Wait()
	if got := r.callCount(); got != 1 {
		t.Errorf("昂贵函数被调用 %d 次, want 1（50 人抢一个 key）", got)
	}
}

// TestConcurrentStress：20 轮 × 8 个并发请求反复轰炸。
func TestConcurrentStress(t *testing.T) {
	r := &recorder{}
	m := New(r.fn)
	defer m.Close()
	urls := urlStream()

	for round := 0; round < 20; round++ {
		var n sync.WaitGroup
		for _, url := range urls {
			n.Add(1)
			go func(url string) {
				defer n.Done()
				if _, err := m.Get(url); err != nil {
					t.Errorf("round %d Get(%q): %v", round, url, err)
				}
			}(url)
		}
		n.Wait()
	}
	// 20 轮 × 8 个请求，key 只有 4 个 → f 只该在第一轮被调用 4 次
	if got := r.callCount(); got != 4 {
		t.Errorf("昂贵函数被调用 %d 次, want 4", got)
	}
}
