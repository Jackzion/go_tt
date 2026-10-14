// 中间件行为测试：用 net/http/httptest 直接调 engine.ServeHTTP，
// 不占端口、不起进程，毫秒级验证洋葱模型的三个核心承诺。
//
//	cd day1-http-base\base4\gee
//	go test -v .
package gee

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMiddlewareOrder 验证执行顺序（洋葱展开）：
// 全局前 → 组前 → handler → 组后 → 全局后
func TestMiddlewareOrder(t *testing.T) {
	r := New()
	var order []string

	r.Use(func(c *Context) {
		order = append(order, "global-before")
		c.Next() // 放行，等里面的人回来
		order = append(order, "global-after")
	})

	v2 := r.Group("/v2")
	v2.Use(func(c *Context) {
		order = append(order, "v2-before")
		c.Next()
		order = append(order, "v2-after")
	})
	v2.GET("/ping", func(c *Context) {
		order = append(order, "handler")
		c.String(200, "pong")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/v2/ping", nil))

	want := "global-before,v2-before,handler,v2-after,global-after"
	if got := strings.Join(order, ","); got != want {
		t.Fatalf("洋葱顺序错误:\n got: %s\nwant: %s", got, want)
	}
	if w.Code != 200 || w.Body.String() != "pong" {
		t.Fatalf("响应不对: code=%d body=%q", w.Code, w.Body.String())
	}
	t.Logf("洋葱展开顺序 ✓ %s", want)
}

// TestMiddlewareGroupIsolation 验证：v2 的中间件绝不会泄漏到根路由。
func TestMiddlewareGroupIsolation(t *testing.T) {
	r := New()
	var hits []string
	r.Use(func(c *Context) { hits = append(hits, "global"); c.Next() })
	v2 := r.Group("/v2")
	v2.Use(func(c *Context) { hits = append(hits, "v2"); c.Next() })
	r.GET("/", func(c *Context) { c.String(200, "root") })
	v2.GET("/x", func(c *Context) { c.String(200, "v2x") })

	// 请求根路由：只有全局中间件
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if got := strings.Join(hits, ","); got != "global" {
		t.Fatalf("根路由只该触发 global，got: %s", got)
	}

	// 请求 v2 路由：全局 + 组，顺序正确
	hits = nil
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/v2/x", nil))
	if got := strings.Join(hits, ","); got != "global,v2" {
		t.Fatalf("v2 路由应触发 global,v2，got: %s", got)
	}
	t.Log("组隔离 ✓ 根路由不沾 v2 中间件，v2 路由按序叠加")
}

// TestFailSkipsHandler 验证 Fail：跳过后续 handler，
// 但外层中间件的收尾代码（c.Next() 之后）仍会执行。
func TestFailSkipsHandler(t *testing.T) {
	r := New()
	afterRan := false
	r.Use(func(c *Context) {
		c.Next()
		afterRan = true // 外层收尾：即使内层 Fail 了也要执行到这里
	})

	v2 := r.Group("/v2")
	v2.Use(func(c *Context) { c.Fail(500, "Internal Server Error") })
	handlerRan := false
	v2.GET("/boom", func(c *Context) {
		handlerRan = true
		c.String(200, "never")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/v2/boom", nil))

	if handlerRan {
		t.Fatal("Fail 之后 handler 不该执行")
	}
	if !afterRan {
		t.Fatal("外层中间件的收尾代码应该执行（洋葱外层永远收尾）")
	}
	if w.Code != 500 || !strings.Contains(w.Body.String(), "Internal Server Error") {
		t.Fatalf("应返回 500 Internal Server Error, got code=%d body=%q", w.Code, w.Body.String())
	}
	t.Log("Fail ✓ 跳过 handler，外层收尾照常，响应 500")
}
