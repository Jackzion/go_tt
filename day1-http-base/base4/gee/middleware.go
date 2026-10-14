package gee

import (
	"log"
	"time"
)

// Logger 是全局日志中间件：记录每个请求的状态码、URI 和总耗时。
// 写法套路：先记时间 → c.Next() 放行（里面所有环节执行完才回到这行之后）→ 收尾记录。
func Logger() HandlerFunc {
	return func(c *Context) {
		t := time.Now()
		c.Next() // 放行：handler 和内层中间件都发生在这一行"背后"
		log.Printf("[%d] %s %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}
