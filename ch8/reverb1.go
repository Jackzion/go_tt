package ch8

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

func ReverbServer() {
	linstener, err := net.Listen("tcp", "localhost:8999")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("reverb server linsten on localhost:8999")
	for {
		conn, err := linstener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handleEcho(conn)
	}
}

func echo(c net.Conn, shout string, delay time.Duration, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Fprintln(c, "\t", strings.ToUpper(shout))
	time.Sleep(delay)
	fmt.Fprintln(c, "\t", shout)
	time.Sleep(delay)
	fmt.Fprintln(c, "\t", strings.ToLower(shout))
}

func handleEcho(conn net.Conn) {
	defer conn.Close()
	var wg sync.WaitGroup

	// 创建一个 channel 传递输入的行
	lines := make(chan string)
	input := bufio.NewScanner(conn)

	// 创建一个 goroutine 持续读取输入
	go func() {
		for input.Scan() {
			lines <- input.Text()
		}
		// 客户端关闭时候，关闭 channel
		close(lines)
	}()

	for {
		// 每次循环重置定时器 10s
		timer := time.NewTimer(10 * time.Second)

		select {
		case line, ok := <-lines:
			if !ok {
				fmt.Println("客户端断开连接")
				wg.Wait() // 等待所有 echo 完成 return
				return
			}
			fmt.Println("收到输入:", line)
			// 保证每次echo都是独立的，互不干扰 ， 否则会排队
			wg.Add(1)
			go echo(conn, line, time.Second, &wg)
		case <-timer.C:
			// 10s 无输入，关闭连接
			fmt.Println("10s 无输入，关闭连接")
			wg.Wait() // 等待所有 echo 完成 return
			return
		}

		// 停止计时器，避免内存泄漏 ，避免读取后时间没到，定时器继续运行
		if !timer.Stop() {
			<-timer.C // 等待定时器到期
		}
	}

}
