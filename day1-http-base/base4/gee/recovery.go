package gee

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
)

// trace 把 message 和"触发它的调用栈"拼成一段可读的调试信息。
//
// 原理：Go 的 panic 会带着调用栈一路向上炸，炸到哪层有 recover 就在哪层被接住。
// runtime.Callers 返回调用栈上每个函数的"程序计数器"（pc），
// 再用 FuncForPC + FileLine 换算成"函数 + 文件:行号"。
func trace(message string) []byte {
	var pcs [32]uintptr
	// skip: 0=Callers 本身, 1=trace, 2=Recovery 里那个 deferred func —— 都与事故现场无关，跳过
	n := runtime.Callers(3, pcs[:])

	var str strings.Builder
	str.WriteString(message + "\nTraceback:")
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return []byte(str.String())
}

// Recovery 是错误恢复中间件：任何一个 handler panic 都不会带走整个进程。
//
// 工作原理：defer + recover——defer 在 panic 展开调用栈时必然执行，
// recover() 接住 panic 值，把它从"杀死进程"降级为"一次普通错误"。
// 外层的 Logger 仍会照常收尾记录 [500]（洋葱模型白送的）。
func Recovery() HandlerFunc {
	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				message := fmt.Sprintf("%s", err)
				log.Printf("%s\n\n", trace(message)) // 日志里带完整堆栈，方便定位
				c.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()

		c.Next()
	}
}
