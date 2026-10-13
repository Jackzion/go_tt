// Package gee 是迷你 Web 框架的第二版：
// Handler 签名从 (w, req) 升级为 (c *Context) —— 所有请求级信息统一由 Context 承载。
package gee

import (
	"log"
	"net/http"
)

// HandlerFunc 是本框架的处理函数类型（签名已换成 Context）。
type HandlerFunc func(*Context)

// Engine 是框架入口，内部是一张路由表。
type Engine struct {
	*RouterGroup // 嵌入 - 继承方法和字段
	router       *router
	groups       []*RouterGroup // store all router group
}

type RouterGroup struct {
	prefix      string        // 路由前缀
	middlewares []HandlerFunc // 中间件
	parent      *RouterGroup  // 父路由组
	engine      *Engine       // 引用指向 Engine，避免循环引用。
}

// New 创建框架实例。
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

// Group 创建路由组，用于批量路由组。
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: prefix,
		parent: group,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

// 与 router 有关的方法交由 Engine 处理，这里只保留 handle 方法
func (group *RouterGroup) addRoute(method, path string, handle HandlerFunc) {
	path = group.prefix + path
	log.Printf("Route %4s - %s", method, path)
	group.engine.router.addRoute(method, path, handle)
}

// GET / POST：把方法拼进 key，防止不同方法撞路径。
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
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
