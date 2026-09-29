package ch8

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"gopl.io/ch5/links"
)

// crawlDone 用于广播取消信号
var crawlDone = make(chan struct{})

func crawlCancelled() bool {
	select {
	case <-crawlDone:
		return true
	default:
		return false
	}
}

// 限制 crawl 并发数 ， 只允许同时对 20 条链接进行访问
var tokens = make(chan struct{}, 20)

// crawl 访问每个链接
func crawl(url string) []string {
	fmt.Println(url)
	tokens <- struct{}{} // acquire a token
	list, err := links.Extract(url)
	<-tokens // release the token
	if err != nil {
		log.Print(err)
	}
	return list
}

func TestCrawl1() {
	worklist := make(chan []string)
	var n int // number of pending sends to worklist

	// start with the initial page
	// 注意要另启动一个goroutine，否则会阻塞主goroutine，导致程序退出 , 消费和生产在一个线程永远是大忌
	n++
	go func() {
		worklist <- []string{"https://gopl.io/"}
	}()

	// crawl the web concurrently
	// 避免重复访问
	seen := make(map[string]bool)

	for ; n > 0; n-- {
		for list := range worklist {
			for _, link := range list {
				if !seen[link] {
					seen[link] = true
					n++ // count this new send
					// 并发访问每个链接
					go func(link string) {
						worklist <- crawl(link)
					}(link)
				}
			}
		}
	}
}

// TestCrawl2 直接使用 20 个 goroutine并发访问每个链接
func TestCrawl2() {
	worklist := make(chan []string)  // lists of URLs, may have duplicates
	unseenLinks := make(chan string) // de-duplicated URLs

	// Add command-line arguments to worklist.
	go func() { worklist <- os.Args[1:] }()

	// Create 20 crawler goroutines to fetch each unseen link.
	for i := 0; i < 20; i++ {
		go func() {
			for link := range unseenLinks {
				foundLinks := crawl(link)
				go func() { worklist <- foundLinks }()
			}
		}()
	}

	// The main goroutine de-duplicates worklist items
	// and sends the unseen ones to the crawlers.
	seen := make(map[string]bool)
	for list := range worklist {
		for _, link := range list {
			if !seen[link] {
				seen[link] = true
				unseenLinks <- link
			}
		}
	}
}

func TestCrawl3() {
	// 爬取前先创建 mirror 目录
	const mirrorDir = "mirror"
	// 最大深度为 3
	const maxDepth = 3

	type Link struct {
		URL   string
		Depth int
	}
	// 创建镜像根目录
	os.MkdirAll(mirrorDir, 0755)
	// 获取起始 url ， 用于限定爬取范围
	startURL, _ := url.Parse(os.Args[1])
	onlyDomain := startURL.Host

	worklist := make(chan []Link)  // lists of URLs, may have duplicates
	unseenLinks := make(chan Link) // de-duplicated URLs

	// Add command-line arguments to worklist.
	go func() { worklist <- []Link{{URL: os.Args[1], Depth: 0}} }()

	// Create 20 crawler goroutines to fetch each unseen link.
	for i := 0; i < 20; i++ {
		go func() {
			for link := range unseenLinks {
				// 保存页面到本地
				savePage(link.URL, onlyDomain, mirrorDir)
				foundLinks := crawl(link.URL)
				// 把结果转换为 Link 类型，深度加 1
				var newLinks []Link
				for _, foundLinks := range foundLinks {
					newLinks = append(newLinks, Link{URL: foundLinks, Depth: link.Depth + 1})
				}
				go func() { worklist <- newLinks }()
			}
		}()
	}

	// The main goroutine de-duplicates worklist items
	// and sends the unseen ones to the crawlers.
	seen := make(map[string]bool)
	for list := range worklist {
		for _, link := range list {
			// 过滤掉深度大于 3 的链接
			if link.Depth > maxDepth {
				continue
			}
			if !seen[link.URL] {
				seen[link.URL] = true
				unseenLinks <- link
			}
		}
	}
}

// urlToFilename 把 URL 转成本地文件路径
func urlToFilename(rawURL, mirrorDir string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	// 拼接：mirrorDir + 域名 + 路径
	filePath := path.Join(mirrorDir, u.Host, u.Path)
	// 如果路径以 / 结尾，说明是目录，加上 index.html
	if strings.HasSuffix(filePath, "/") {
		filePath += "index.html"
	}

	return filePath

}

// 把原始链接改写为本地镜像
func rewriteLinks(content []byte, baseURL *url.URL, mirrorDir string) []byte {
	html := string(content)
	// 把 https://golang.org/ 替换成 本地相对路径
	origin := baseURL.Scheme + "://" + baseURL.Host
	html = strings.ReplaceAll(html, origin, "")
	return []byte(html)
}

func savePage(rawURL, onlyDomain, mirrorDir string) {
	if crawlCancelled() {
		return
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return
	}

	// 1. 用 http.NewRequest 创建请求，支持取消
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		log.Print(err)
		return
	}
	req.Cancel = crawlDone
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if crawlCancelled() {
			return
		}
		log.Print(err)
		return
	}
	defer resp.Body.Close()

	// 只处理 HTML 页面
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return
	}

	// 2. 读取内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return
	}

	// 3. 改写链接（绝对路径 → 相对路径）
	body = rewriteLinks(body, u, mirrorDir)

	// 4. 计算本地文件路径
	filePath := urlToFilename(rawURL, mirrorDir)

	// 5. 创建目录并写入文件
	os.MkdirAll(path.Dir(filePath), 0755)
	err = os.WriteFile(filePath, body, 0644)
	if err != nil {
		log.Print(err)
		return
	}
	fmt.Printf("Saved: %s → %s\n", rawURL, filePath)
}

// TestCrawlCancel 支持取消的并发爬虫（练习 8.10）
func TestCrawlCancel() {
	const maxDepth = 3

	type Link struct {
		URL   string
		Depth int
	}

	// 监听取消：按回车触发
	go func() {
		os.Stdin.Read(make([]byte, 1))
		fmt.Println("收到取消信号，正在停止...")
		close(crawlDone)
	}()

	// 起始 URL
	startURL := "https://gopl.io/"
	if len(os.Args) > 1 {
		startURL = os.Args[1]
	}

	worklist := make(chan []Link)
	unseenLinks := make(chan Link)

	go func() {
		worklist <- []Link{{URL: startURL, Depth: 0}}
	}()

	// 20 个爬虫 goroutine
	for i := 0; i < 20; i++ {
		go func() {
			for link := range unseenLinks {
				if crawlCancelled() {
					return
				}
				// 用 http.NewRequest 发请求，支持取消
				req, err := http.NewRequest("GET", link.URL, nil)
				if err != nil {
					log.Print(err)
					continue
				}
				req.Cancel = crawlDone
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					if crawlCancelled() {
						return
					}
					log.Print(err)
					continue
				}
				resp.Body.Close()

				// 提取链接
				fmt.Println(link.URL)
				foundLinks, err := links.Extract(link.URL)
				if err != nil {
					log.Print(err)
					continue
				}
				var newLinks []Link
				for _, l := range foundLinks {
					if !crawlCancelled() {
						newLinks = append(newLinks, Link{URL: l, Depth: link.Depth + 1})
					}
				}
				go func() { worklist <- newLinks }()
			}
		}()
	}

	// main 去重 + 深度限制
	seen := make(map[string]bool)
	var n int
	n++
loop:
	for list := range worklist {
		n--
		for _, link := range list {
			if crawlCancelled() {
				break loop
			}
			if link.Depth > maxDepth {
				continue
			}
			if !seen[link.URL] {
				seen[link.URL] = true
				n++
				unseenLinks <- link
			}
		}
		if n == 0 {
			break loop
		}
	}
	close(unseenLinks)
	fmt.Printf("完成！共爬取 %d 个页面\n", len(seen))
}

// TestMirroredFetch 并发请求多个 URL，第一个返回后取消其余请求（8.4.4 mirroredQuery 变种）
func TestMirroredFetch(urls ...string) string {
	if len(urls) == 0 {
		return ""
	}

	// 每个请求用独立的 cancel channel
	cancel := make(chan struct{})
	type result struct {
		url  string
		body string
		err  error
	}
	ch := make(chan result)

	// 为每个 URL 启动一个 goroutine 发送请求
	for _, url := range urls {
		go func(url string) {
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				ch <- result{url: url, err: err}
				return
			}
			req.Cancel = cancel // 绑定取消 channel

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				ch <- result{url: url, err: err}
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				ch <- result{url: url, err: err}
				return
			}
			ch <- result{url: url, body: string(body)}
		}(url)
	}

	// 取第一个返回的结果，然后取消其余请求
	first := <-ch
	close(cancel) // 广播取消：其余请求立刻中断
	// 排空其余 goroutine（避免它们永远阻塞在 ch <- 上）
	for i := 1; i < len(urls); i++ {
		<-ch
	}

	if first.err != nil {
		fmt.Printf("fetch 失败: %s, error: %v\n", first.url, first.err)
		return ""
	}
	fmt.Printf("最快的响应来自: %s, 长度: %d bytes\n", first.url, len(first.body))
	return first.body
}
