package gee

import (
	"net/http"
	"strings"
)

type router struct {
	// 静态路由快路径：整条路径精确注册过的，直接查表（method-path → handler）
	handlers map[string]HandlerFunc
	// 动态路由：前缀树（支持 :name 和 *filepath）
	roots map[string]*node
}

// roots key eg, roots['GET'] roots['POST']
// handlers key eg, handlers['GET-/p/:lang/doc'], handlers['POST-/p/book']
func newRouter() *router {
	return &router{
		handlers: make(map[string]HandlerFunc),
		roots:    make(map[string]*node),
	}
}

func (r *router) addRoute(method, path string, handle HandlerFunc) {
	key := method + "-" + path
	r.handlers[key] = handle
	// 初始化 METHOD的前缀树
	if _, ok := r.roots[method]; !ok {
		r.roots[method] = &node{}
	}

	r.roots[method].insert(path, parsePath(path), 0)
}

// getRoute 查找匹配的树节点，并提取动态参数。
// 返回 (节点, 参数表)；没找到时节点为 nil。
func (r *router) getRoute(method, path string) (*node, map[string]string) {
	params := make(map[string]string)
	parts := parsePath(path)
	root, ok := r.roots[method]
	if !ok {
		return nil, params
	}

	// 查找匹配的树节点
	n := root.search(parts, 0)
	if n == nil {
		return nil, params
	}

	// 参数提取：把"模式的各段"和"请求的各段"逐层对齐：
	//   /hello/:name  vs  /hello/geektutu  → name=geektutu
	//   /assets/*fp   vs  /assets/css/a.css → filepath=css/a.css（* 吞掉剩余所有段）
	patternParts := parsePath(n.pattern)
	for i, part := range patternParts {
		if i >= len(parts) {
			break
		}
		switch part[0] {
		case ':':
			params[part[1:]] = parts[i]
		case '*':
			params[part[1:]] = strings.Join(parts[i:], "/")
			return n, params // * 是终点，后面没有要对齐的了
		}
	}
	return n, params
}

func (r *router) handle(c *Context) {
	// ① 静态快路径：整条路径精确注册过，不用爬树
	if handler, ok := r.handlers[c.Method+"-"+c.Path]; ok {
		handler(c)
		return
	}

	// ② 爬树：找节点 + 提取动态参数
	n, params := r.getRoute(c.Method, c.Path)
	if n != nil {
		c.Params = params
		key := c.Method + "-" + n.pattern // 用模式（而非请求路径）反查 handler
		r.handlers[key](c)
		return
	}

	c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
}
