package ch8

import (
	"log"
	"os"
	"sync"

	"gopl.io/ch8/thumbnail"
)

// makes thumbnails of the specified files in parallel.
func makeThumbnails(filenames []string) int64 {

	type item struct {
		filename string
		err      error
	}
	var wg sync.WaitGroup // number of working goroutines
	// make a channel of item
	sizes := make(chan int64)
	for _, f := range filenames {
		wg.Add(1)
		go func(f string) {
			// 完成会减少一个goroutine告知wg.Wait
			defer wg.Done()
			thumb, err := thumbnail.ImageFile(f)
			if err != nil {
				log.Println(err)
				return
			}
			// 计算文件大小
			info, _ := os.Stat(thumb)
			// 发送文件大小到sizes channel
			sizes <- info.Size()

		}(f)
	}
	// closer
	go func() {
		wg.Wait()
		close(sizes)
	}()

	// 计算总大小
	var totalSize int64
	for size := range sizes {
		totalSize += size
	}

	return totalSize
}
