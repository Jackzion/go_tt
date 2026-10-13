// 用自己写的 gee 框架重写 base1 的功能。
// 业务代码的写法已经和 gin 几乎一样了：
//
//	engine := gee.New()
//	engine.GET("/path", handler)
//	engine.Run(":9999")
package main

import (
	"fmt"
	"net/http"
	"time"

	"gee"
)

func main() {
	engine := gee.New()

	engine.GET("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
	})

	engine.GET("/hello", func(w http.ResponseWriter, req *http.Request) {
		for k, v := range req.Header {
			fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
		}
	})

	engine.GET("/time", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Time = %v\n", time.Now())
	})

	engine.Run(":9999")
}
