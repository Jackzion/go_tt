// 针对 memo1（无并发保护版）的系统性测试：
//   - TestSequential：鉴定"缓存效果"——第二次起是否真的变快、昂贵函数是否少调用
//   - TestConcurrent：鉴定"并发缺陷"——8 个 goroutine 同时用会怎样（预期：崩/挂）
//   - TestNetwork  ：可选，用真实 httpGetBody 走外网（默认跳过）
//
// 运行方式：
//
//	go test -v -run TestSequential ./ch9/memo1   顺序测试：PASS
//	go test -v -run TestConcurrent ./ch9/memo1   并发测试：崩溃或 FAIL（这就是 memo1 的病）
//	go test -race -v ./ch9/memo1                 竞争检查器给出精确报告
//	MEMO_NET_TEST=1 go test -v -run TestNetwork ./ch9/memo1  真实网络版
package memo

import (
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

const slowCost = 50 * time.Millisecond // 模拟"昂贵操作"的耗时

// urlStream 模拟书里的 incomingURLs：4 个不同 URL，每个出现两次（有重复）。
func urlStream() []string {
	return []string{
		"https://golang.org",
		"https://godoc.org",
		"https://play.golang.org",
		"http://gopl.io",
		"https://golang.org",      // 重复
		"https://godoc.org",       // 重复
		"https://play.golang.org", // 重复
		"http://gopl.io",          // 重复
	}
}

// recorder 包装那个"昂贵函数"，统计调用次数。
// 注意：它自己带锁 —— 测量工具必须自身线程安全，不能给被测对象添乱。
type recorder struct {
	mu    sync.Mutex
	calls int
	keys  []string
}

// httpGetBody 是一个昂贵的函数，用于获取 HTTP 响应体。
// 它的缓存版本就是 Memo。
func httpGetBody(url string) (any, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (r *recorder) fn(key string) (any, error) {
	r.mu.Lock()
	r.calls++
	r.keys = append(r.keys, key)
	r.mu.Unlock()
	time.Sleep(slowCost) // 模拟网络请求的耗时
	return []byte("data:" + key), nil
}

func (r *recorder) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// TestSequential 鉴定缓存效果（对应书里第一段测试输出）。
// 期望：8 次 Get 只触发 4 次昂贵调用；重复 URL 秒回。
func TestSequential(t *testing.T) {
	r := &recorder{}
	m := New(r.fn)

	for i, url := range urlStream() {
		start := time.Now()
		value, err := m.Get(url)
		if err != nil {
			t.Fatalf("Get(%q): %v", url, err)
		}
		elapsed := time.Since(start)
		t.Logf("%s, %s, %d bytes", url, elapsed.Truncate(time.Microsecond), len(value.([]byte)))

		// ① 值必须正确
		want := "data:" + url
		if got := string(value.([]byte)); got != want {
			t.Errorf("第%d次 Get(%q) = %q, want %q", i, url, got, want)
		}
		// ② 第二次遇到同一 URL 必须走缓存（50ms 的慢操作，命中应远小于 10ms）
		if i >= 4 && elapsed > 10*time.Millisecond {
			t.Errorf("第%d次 Get(%q) 花了 %v —— 应该是缓存命中，不该这么慢", i, url, elapsed)
		}
	}

	// ③ 核心指标：8 次请求、4 个不同 URL，昂贵函数只该被调用 4 次
	if got := r.callCount(); got != 4 {
		t.Errorf("昂贵函数被调用 %d 次, want 4（重复的 URL 没走缓存！）", got)
	}
}

// TestConcurrent 鉴定并发缺陷（对应书里第二段测试代码）。
// 期望（假如 memo 是并发安全的）：8 个 goroutine 同时请求，昂贵函数仍只调用 4 次，
// 且每个 goroutine 都拿到属于自己 URL 的正确值。
// 现实（memo1 没有任何保护）：可能直接 fatal crash，或者调用次数 > 4，或者值串了。
func TestConcurrent(t *testing.T) {
	r := &recorder{}
	m := New(r.fn)
	urls := urlStream()

	results := make([]any, len(urls))
	errs := make([]error, len(urls))

	var n sync.WaitGroup
	for i, url := range urls {
		n.Add(1)
		go func(i int, url string) { // 书里是 go func(url string)，这里多传个下标方便对账
			defer n.Done()
			results[i], errs[i] = m.Get(url) // ⚠️ 8 个 goroutine 同时读写同一张 map
		}(i, url)
	}
	n.Wait()

	// ① 每个 goroutine 拿到的值必须和自己请求的 URL 对得上（不能张冠李戴）
	for i, url := range urls {
		if errs[i] != nil {
			t.Errorf("Get(%q): %v", url, errs[i])
			continue
		}
		want := "data:" + url
		if got := string(results[i].([]byte)); got != want {
			t.Errorf("Get(%q) 返回了 %q, want %q —— 值串台了！", url, got, want)
		}
	}

	// ② 并发去重：4 个不同的 URL，昂贵函数只该被调用 4 次
	got := r.callCount()
	if got != 4 {
		t.Errorf("昂贵函数被调用 %d 次, want 4 —— 有重复下载（意外的 cache miss）", got)
	}
	t.Logf("昂贵函数实际被调用 %d 次", got)
}
