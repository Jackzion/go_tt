// day6：服务端渲染。
// 静态文件：r.Static 把磁盘目录映射到 URL 前缀（复用 *filepath 通配路由）。
// 模板渲染：LoadHTMLGlob 预加载模板，c.HTML(code, 模板名, 数据) 渲染输出。
package main

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"gee"
)

type student struct {
	Name string
	Age  int8
}

// FormatAsDate 是自定义模板函数：time.Time → "年-月-日" 字符串。
func FormatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

func main() {
	r := gee.New()
	// 注意注册顺序：Logger 在外，Recovery 在内——
	// panic 被 Recovery 接住后 Fail 500，返回途中 Logger 照常记录 [500]。
	r.Use(gee.Logger(), gee.Recovery())
	// 注意顺序：先装插件（自定义函数），再加载引用了插件的模板
	r.SetFuncMap(template.FuncMap{
		"FormatAsDate": FormatAsDate,
	})
	r.LoadHTMLGlob("templates/*")
	r.Static("/assets", "./static")

	stu1 := &student{Name: "Geektutu", Age: 20}
	stu2 := &student{Name: "Jack", Age: 22}
	r.GET("/", func(c *gee.Context) {
		c.HTML(http.StatusOK, "css.tmpl", nil)
	})
	r.GET("/students", func(c *gee.Context) {
		c.HTML(http.StatusOK, "arr.tmpl", gee.H{
			"title":  "gee",
			"stuArr": [2]*student{stu1, stu2},
		})
	})

	r.GET("/date", func(c *gee.Context) {
		c.HTML(http.StatusOK, "custom_func.tmpl", gee.H{
			"title": "gee",
			"now":   time.Date(2019, 8, 17, 0, 0, 0, 0, time.UTC),
		})
	})

	// 故意的事故现场：names[100] 数组越界 → panic
	// 预期：Recovery 接住 → 返回 500 Internal Server Error，进程存活，其他路由照常
	r.GET("/panic", func(c *gee.Context) {
		names := []string{"geektutu"}
		c.String(http.StatusOK, names[100]) // index out of range → panic!
	})

	r.Run(":9999")
}
