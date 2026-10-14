// Package gee 是迷你 Web 框架的第二版：
// Handler 签名从 (w, req) 升级为 (c *Context) —— 所有请求级信息统一由 Context 承载。
package gee

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
)

// HandlerFunc 是本框架的处理函数类型（签名已换成 Context）。
type HandlerFunc func(*Context)

// Engine 是框架入口，内部是一张路由表。
type Engine struct {
	*RouterGroup  // 嵌入 - 继承方法和字段
	router        *router
	groups        []*RouterGroup     // store all router group
	htmlTemplates *template.Template // 模板集：启动时 ParseGlob 一次性加载进内存
	funcMap       template.FuncMap   // 自定义模板函数（如 FormatAsDate）
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

// Use 注册中间件。
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
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

// SetFuncMap 注册自定义模板渲染函数（须在 LoadHTMLGlob 之前调用）。
func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

// LoadHTMLGlob 把模板文件一次性加载进内存（之后每次渲染不再读磁盘）。
func (engine *Engine) LoadHTMLGlob(pattern string) {
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}

// createStaticHandler 把"磁盘目录 fs"挂到路由前缀 absolutePath 上。
// 复用 *filepath 通配路由拿到相对路径，再交给标准库的 http.FileServer 搬运。
func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(group.prefix, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filepath")
		// 先探路：文件不存在/没权限 → 404，绝不把磁盘错误漏给客户端
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Req) // 真正的搬运交给标准库
	}
}

// Static 对外 API：把磁盘上的 root 目录映射到 URL 前缀 relativePath。
// 例：r.Static("/assets", "./static")
//     访问 /assets/js/a.js → 返回 ./static/js/a.js
func (group *RouterGroup) Static(relativePath string, root string) {
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	group.GET(urlPattern, handler)
}

// ServeHTTP —— 框架心脏。
// 每个请求进来，先造一个 Context（百宝箱），查表后交给 handler。
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var middlewares []HandlerFunc
	for _, group := range engine.groups {
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c := NewContext(w, req)
	c.engine = engine // 让 Context 够得着 Engine 里的模板集
	c.handlers = middlewares
	engine.router.handle(c)
}

// Run 启动服务。
func (engine *Engine) Run(addr string) {
	http.ListenAndServe(addr, engine)
}
