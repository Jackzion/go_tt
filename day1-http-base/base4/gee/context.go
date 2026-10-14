package gee

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// H 是 map[string]interface{} 的别名，写 JSON 响应时省点手指。
type H map[string]interface{}

// Context 是一次请求的"百宝箱"：
//   - 封装 Request / ResponseWriter，把高频操作（查询参数、构造响应）收拢成短方法
//   - 同时承载请求级状态（Path、将来的动态路由参数、中间件产生的信息）
//   - 随请求创建、随请求销毁，生命周期与一次会话完全对齐
type Context struct {
	Writer     http.ResponseWriter
	Req        *http.Request
	Path       string // 当前请求路径（main.go 里的 c.Path 靠它）
	Method     string
	StatusCode int
	Params     map[string]string // 动态路由参数，如 :name → "geektutu"
	// 中间件
	handlers []HandlerFunc
	index    int
}

// Param 读取动态路由参数：/hello/:name 匹配 /hello/geektutu 时，c.Param("name") == "geektutu"
func (c *Context) Param(key string) string {
	return c.Params[key]
}

// -------- 读请求 --------

// Query 读取 URL 查询参数：/hello?name=geektutu → c.Query("name") == "geektutu"
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

// PostForm 读取表单参数（application/x-www-form-urlencoded）。
func (c *Context) PostForm(key string) string {
	_ = c.Req.ParseForm() // 解析失败时 FormValue 返回空串，可接受
	return c.Req.FormValue(key)
}

// -------- 写响应 --------

// Status 设置状态码（必须在第一次 Write 前调用）。
func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

// SetHeader 设置响应头（同样必须在第一次 Write 前调用）。
func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

// String 发送 text/plain 响应。
func (c *Context) String(code int, format string, values ...any) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

// JSON 发送 application/json 响应 —— 对比封装前的手写版，感受一下。
func (c *Context) JSON(code int, obj any) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	if err := json.NewEncoder(c.Writer).Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
	}
}

func (c *Context) Next() {
	c.index++
	for ; c.index < len(c.handlers); c.index++ {
		c.handlers[c.index](c)
	}
}

// Fail 紧急叫停：跳过链上剩余的所有中间件和 handler，直接写错误响应。
// 注意：外层正在等待收尾的中间件（如写在 c.Next() 之后的 Logger）仍会执行——
// 这正是洋葱模型的价值：错误也能被外层统一记录。
func (c *Context) Fail(code int, msg string) {
	c.index = len(c.handlers) // 游标拨到链尾，Next 循环直接结束
	c.String(code, msg)
}

func NewContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
		index:  -1,
	}
}

// Data 发送任意字节（下载文件、图片等场景）。
func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

// HTML 发送 text/html 响应。
func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Writer.Write([]byte(html))
}
