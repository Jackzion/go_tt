// base4：用 Context 版 gee 框架。
// Handler 签名变成 func(*gee.Context)，响应构造收敛为 c.HTML/String/JSON。
package main

import (
	"net/http"

	"gee"
)

func main() {
	r := gee.New()
	r.GET("/", func(c *gee.Context) {
		c.HTML(http.StatusOK, "<h1>Hello Gee</h1>")
	})
	r.GET("/hello", func(c *gee.Context) {
		// expect /hello?name=geektutu
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
	})

	r.POST("/login", func(c *gee.Context) {
		c.JSON(http.StatusOK, gee.H{
			"username": c.PostForm("username"),
			"password": c.PostForm("password"),
		})
	})

	// 动态路由：:name 匹配一段，*filepath 匹配剩余全部
	r.GET("/hello/:name", func(c *gee.Context) {
		// expect /hello/geektutu → hello geektutu, you're at /hello/geektutu
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
	})
	r.GET("/assets/*filepath", func(c *gee.Context) {
		// expect /assets/css/geektutu.css → filepath=css/geektutu.css
		c.JSON(http.StatusOK, gee.H{"filepath": c.Param("filepath")})
	})

	r.Run(":9999")
}
