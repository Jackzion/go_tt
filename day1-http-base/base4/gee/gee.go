// Package gee 是迷你 Web 框架的第二版：
// Handler 签名从 (w, req) 升级为 (c *Context) —— 所有请求级信息统一由 Context 承载。
package gee

import (
	"net/http"
)

// HandlerFunc 是本框架的处理函数类型（签名已换成 Context）。
type HandlerFunc func(*Context)

// Engine 是框架入口，内部是一张路由表。
type Engine struct {
	router *router
}

// New 创建框架实例。
func New() *Engine {
	return &Engine{router: newRouter()}
}

// GET / POST：把方法拼进 key，防止不同方法撞路径。
func (engine *Engine) GET(pattern string, handler HandlerFunc) {
	engine.router.addRoute("GET", pattern, handler)
}

func (engine *Engine) POST(pattern string, handler HandlerFunc) {
	engine.router.addRoute("POST", pattern, handler)
}

// ServeHTTP —— 框架心脏。
// 每个请求进来，先造一个 Context（百宝箱），查表后交给 handler。
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := NewContext(w, req)
	engine.router.handle(c)
}

// Run 启动服务。
func (engine *Engine) Run(addr string) {
	http.ListenAndServe(addr, engine)
}
