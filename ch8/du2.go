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
func TestDu2 () { // 1) 参数与环境准备 flag.Parse()
	roots := flag.Args() if len (roots) == 0 {
		roots = [] string { "." }
	} // 2) 令牌信号量，限制并发目录遍历数量 const maxConcurrentDirs = 50 sema := make ( chan struct {}, maxConcurrentDirs) // 3) 用于等待所有 goroutine 完成的 WaitGroup var wg sync.WaitGroup // 4) 用一个 channel 来分发待处理的目录（简化起见，使用无缓冲 channel） todo := make ( chan string , 100 ) // 5) 启动多个 worker，负责从 todo 队列取出目录并遍历 workerCount := maxConcurrentDirs for i := 0 ; i < workerCount; i++ {
		wg.Add( 1 ) go func () { defer wg.Done() for d := range todo {
				walkDir(d, fileSizes, sema, &wg, todo)
			}
		}()
	} // 6) 将初始根目录放进 todo 队列（注意：不要阻塞地放进无缓冲 channel） for _, r := range roots {
		wg.Add( 1 ) // 这里选择不阻塞地放入 todo，防止在 wg.Wait 之前就死锁 go func (r string ) { defer wg.Done() // 尝试直接投递，若队列满则阻塞在 goroutine 内也不影响主线程 todo <- r
		}(r)
	} // 7) 等待所有任务完成并关闭 fileSizes（假设你用 fileSizes 收集大小） // 采用一个 goroutine 来等待所有 worker 完成后再关闭 fileSizes fileSizes := make ( chan int64 , 1000 ) go func () { // 等待所有 worker 结束 wg.Wait() close (fileSizes)
	}() // 8) 收集结果并打印 var nfiles, nbytes int64 tick := time.Tick( 500 * time.Millisecond)
Loop: for { select { case size, ok := <-fileSizes: if !ok { break Loop
			}
			nfiles++
			nbytes += size case <-tick:
			printDiskUsage(nfiles, nbytes)
		}
	}
	printDiskUsage(nfiles, nbytes)
} 
func walkDir (dir string , fileSizes chan <- int64 , sema chan struct {}, wg *sync.WaitGroup, todo chan string ) { // 获取令牌，控制并发 sema <- struct {}{} // 令牌用尽时会阻塞 defer func () { <-sema }()

	entries, err := ioutil.ReadDir(dir) if err != nil {
		fmt.Fprintln(os.Stderr, err) return } for _, e := range entries { if e.IsDir() { // 直接把子目录投入 todo，避免在当前 goroutine 中递归造成深层嵌套 // 但要确保 wg.Add(1) 与 go 的调用点成对 wg.Add( 1 ) go func () { defer wg.Done()
				todo <- filepath.Join(dir, e.Name())
			}()
		} else { // 发送文件大小到收集通道 fileSizes <- e.Size()
		}
	}
}

func printDiskUsage(nfiles, nbytes int64) {
	switch {
	case nbytes >= 1e9:
		fmt.Printf("%d files  %.1f GB\n", nfiles, float64(nbytes)/1e9)
	case nbytes >= 1e6:
		fmt.Printf("%d files  %.1f MB\n", nfiles, float64(nbytes)/1e6)
	case nbytes >= 1e3:
		fmt.Printf("%d files  %.1f KB\n", nfiles, float64(nbytes)/1e3)
	default:
		fmt.Printf("%d files  %d bytes\n", nfiles, nbytes)
	}
}

// walkDir recursively walks the file tree rooted at dir
// and sends the size of each found file on fileSizes.
func walkDir(dir string, n *sync.WaitGroup, fileSizes chan<- int64, sema chan struct{}) {
	defer n.Done()
	for _, entry := range dirents(dir) {
		if entry.IsDir() {
			subdir := filepath.Join(dir, entry.Name())
			sema <- struct{}{} // acquire a token
			n.Add(1)
			go func() {
				walkDir(subdir, n, fileSizes, sema)
				<-sema // release the token
			}()
		} else {
			fileSizes <- entry.Size()
		}
	}
}

// dirents returns the entries of directory dir.
func dirents(dir string) []os.FileInfo {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "du1: %v\n", err)
		return nil
	}
	return entries
}
