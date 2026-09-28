package ch8

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var verbose = flag.Bool("v", false, "show verbose progress messages")

// abort goroutine
var done = make(chan struct{})

func cancelled() bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}

// TestDu3 每隔一段时间显示 root 目录下的目录大小
func TestDu3() {

	// 监听 abort goroutine
	go func() {
		os.Stdin.Read(make([]byte, 1))
		close(done)
	}()

	// 解析命令行参数
	flag.Parse()
	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}

	// 定时打印进度
	var tick <-chan time.Time
	if *verbose {
		tick = time.Tick(500 * time.Millisecond)
	}

	// 收集文件大小的通道
	fileSizes := make(chan int64)

	// 启动目录遍历
	go func() {
		var wg sync.WaitGroup
		for _, root := range roots {
			wg.Add(1)
			go walkDir3(root, &wg, fileSizes)
		}
		wg.Wait()
		close(fileSizes)
	}()

	// 定时打印 + 最终结果
	var nfiles, nbytes int64
loop:
	for {
		select {
		case <-done:
			for range fileSizes {
				// 等待所有文件大小被处理
			}
			// 确保所有 goroutine 都退出
			panic("checking goroutines") // 触发栈 dump
		case size, ok := <-fileSizes:
			if !ok {
				break loop
			}
			nfiles++
			nbytes += size
		case <-tick:
			PrintDiskUsage(nfiles, nbytes)
		}
	}
	PrintDiskUsage(nfiles, nbytes)
}

// walkDir3 递归遍历目录，把文件大小发到 channel
func walkDir3(dir string, wg *sync.WaitGroup, fileSizes chan<- int64) {
	defer wg.Done()
	if cancelled() {
		return
	}
	for _, entry := range dirents3(dir) {
		if entry.IsDir() {
			subdir := filepath.Join(dir, entry.Name())
			wg.Add(1)
			go walkDir3(subdir, wg, fileSizes)
		} else {
			fileSizes <- entry.Size()
		}
	}
}

func dirents3(dir string) []os.FileInfo {
	entries, err := ioutil.ReadDir(dir)
	if cancelled() {
		return nil
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "du3: %v\n", err)
		return nil
	}
	return entries
}

func PrintDiskUsage(nfiles, nbytes int64) {
	fmt.Printf("%d files, %d bytes\n", nfiles, nbytes)
}
