package ch8

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

func TestClockWall() {
	// 解析参数 ， 收集所有时钟
	type Clock struct {
		name string
		host string
	}
	var clocks []Clock
	flag.Parse()

	// 解析命令行参数 , 每个参数格式为 name=host ， save to clocks slice
	for _, arg := range flag.Args() {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			fmt.Printf("invalid arguments: %s\n", arg)
			continue
		}
		clocks = append(clocks, Clock{name: parts[0], host: parts[1]})
	}

	// 锁
	var mu sync.Mutex
	// 共享数据，每个时区最新时间
	zones := make(map[string]string)

	// 为每个服务器启动一个 goroutine ， 读取时间并更新 zones map
	var wg sync.WaitGroup
	for _, c := range clocks {
		wg.Add(1)
		go func(name, host string) {
			defer wg.Done()
			// 链接服务器
			conn, err := net.Dial("tcp", host)
			if err != nil {
				fmt.Printf("failed to connect to %s: %v\n", host, err)
				return
			}
			defer conn.Close()

			// 读一行时间
			for {
				reader := bufio.NewReader(conn)
				line, err := reader.ReadString('\n')
				if err != nil {
					fmt.Printf("failed to read from %s: %v\n", host, err)
					return // 断开了
				}
				// 多线程锁
				mu.Lock()
				zones[name] = strings.TrimSpace(line)
				mu.Unlock()
			}

		}(c.name, c.host)
	}

	// 4. 主循环：清屏打印表格
	for {
		mu.Lock()
		fmt.Print("\033[H\033[2J") // 清屏
		fmt.Println("=== Clock Wall ===")
		for _, c := range clocks {
			fmt.Printf("%-10s: %s\n", c.name, zones[c.name])
		}
		mu.Unlock()
		time.Sleep(1 * time.Second)
	}

}
