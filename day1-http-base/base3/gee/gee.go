// Package gee 是一个迷你 Web 框架的雏形。
//
// 核心思想：自己实现 http.Handler 接口（ServeHTTP 方法），
// 把所有请求先接到自己的路由表上查表，查到了就调用注册的处理函数。
// 这就是 gin/echo 等框架内部的第一层原理。
package gee

import (
	"fmt"
	"net/http"
)

// HandlerFunc 是本框架的处理函数类型。
// 直接复用标准库的形状，让标准库的生态无缝可用。
type HandlerFunc func(http.ResponseWriter, *http.Request)

// Engine 是框架入口，内部就是一张路由表。
// key = 方法 + 路径（如 "GET/hello"），value = 处理函数。
type Engine struct {
	router map[string]HandlerFunc
}

// New 创建框架实例。
func New() *Engine {
	return &Engine{router: make(map[string]HandlerFunc)}
}

// addRoute 注册一条路由（小写，包内使用）。
func (engine *Engine) addRoute(pattern string, handler HandlerFunc) {
	engine.router[pattern] = handler
}

// GET / POST 是对外的语法糖：把方法拼进 key，防止不同方法撞路径。
func (engine *Engine) GET(pattern string, handler HandlerFunc) {
	engine.addRoute("GET"+pattern, handler)
}

func (engine *Engine) POST(pattern string, handler HandlerFunc) {
	engine.addRoute("POST"+pattern, handler)
}

// ServeHTTP —— 整个框架的心脏。
// 只要 Engine 实现了这个方法，它就是一个合格的 http.Handler，
// 可以直接交给 http.ListenAndServe 使用。
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := req.Method + req.URL.Path
	if handler, ok := engine.router[key]; ok {
		handler(w, req) // 查到 → 执行注册的处理函数
	} else {
		fmt.Fprintf(w, "404 NOT FOUND: %s\n", req.URL.Path) // 没查到 → 兜底
	}
}

// Run 启动服务。注意传入的是 engine 本身（它满足 http.Handler 接口）。
func (engine *Engine) Run(addr string) {
	http.ListenAndServe(addr, engine)
}
