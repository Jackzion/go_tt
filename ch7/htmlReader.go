package ch7

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/net/html"
)

// HtmlReader 结构体用于读取 HTML 内容
type HtmlReader struct {
	html []byte
	i    int
}

// Read 方法实现了 io.Reader 接口，用于读取 html 内容
func (r *HtmlReader) Read(p []byte) (n int, err error) {
	// 存储索引大于 byte 数组长度时，返回 EOF 错误 , 表示读完
	if r.i >= len(r.html) {
		return 0, io.EOF
	}
	// 复制 html 内容到 p 中，返回复制的字节数
	n = copy(p, r.html[r.i:])
	// 更新索引
	r.i += n
	return
}

// NewHtmlReader 函数用于创建一个新的 HtmlReader 实例
func NewHtmlReader(html string) *HtmlReader {
	return &HtmlReader{html: []byte(html), i: 0}
}

func TestHtmlReader() {
	doc, err := html.Parse(NewHtmlReader("<html><body><h1>Hello, World!</h1></body></html>"))
	if err != nil {
		fmt.Println("Error parsing HTML:", err)
		os.Exit(1)
	}
	fmt.Printf("Parsed HTML document: %#v", doc)
}
