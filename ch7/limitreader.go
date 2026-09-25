package ch7

import (
	"fmt"
	"io"
	"strings"
)

type LimitReader struct {
	r io.Reader // underlying reader
	n int64     // max bytes remaining
}

func (l *LimitReader) Read(p []byte) (n int, err error) {
	if l.n <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.n {
		p = p[0:l.n]
	}
	n, err = l.r.Read(p)
	l.n -= int64(n)
	return
}

func NewLimitReader(r io.Reader, n int64) *LimitReader {
	return &LimitReader{r, n}
}

func TestLimitReader() {
	r := strings.NewReader("Hello, World! This is a long string.")
	lr := NewLimitReader(r, 5) // 只允许读 5 个字节
	data, err := io.ReadAll(lr)
	fmt.Println(string(data), err)
	// 输出: Hello <nil>
}
