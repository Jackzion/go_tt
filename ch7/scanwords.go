package ch7

import (
	"bufio"
	"bytes"
	"fmt"
)

type WordsCount int
type LinesCount int

// 使用来自ByteCounter的思路，实现一个针对单词和行数的计数器。你会发现bufio.ScanWords非常的有用。

func (s *WordsCount) Write(p []byte) (int, error) {
	var sc = bufio.NewScanner(bytes.NewReader(p))
	sc.Split(bufio.ScanWords)
	for sc.Scan() {
		*s++
	}
	return int(*s), nil
}

func (s *LinesCount) Write(p []byte) (int, error) {
	var sc = bufio.NewScanner(bytes.NewReader(p))
	sc.Split(bufio.ScanLines)
	for sc.Scan() {
		*s++
	}
	return int(*s), nil
}

func TestScanWords() {

	var wc WordsCount
	var lc LinesCount
	wc.Write([]byte("hello world"))
	lc.Write([]byte("hello\nworld\n"))
	fmt.Println(wc, lc)
	fmt.Fprintf(&wc, "Hello, %s", "dugulp")
	fmt.Println(wc, lc)
}