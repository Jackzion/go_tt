package ch8

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"
)

type client chan<- string

type clientInfo struct {
	ch   client
	name string
}

var (
	entering = make(chan clientInfo) // 管道里传输的是"对讲机"（client类型）
	leaving  = make(chan clientInfo) // 管道里传输的是"对讲机"
	messages = make(chan string)     // 管道里传输的是"纸条"（字符串）
)

func TestChat() {
	// 启动服务器
	listener, err := net.Listen("tcp", ":8999")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	// 广播
	go broadcaster()
	// 处理客户端连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handleClient(conn)
	}
}

// handleClient 处理客户端连接
func handleClient(conn net.Conn) {
	// 创建客户端通道
	ch := make(chan string)
	go clientWriter(conn, ch)

	who := conn.RemoteAddr().String()
	ch <- "You are " + who
	// 加入广播
	entering <- clientInfo{ch, who}
	messages <- who + " joined"

	// 启动一个 goroutine 监控空闲超时
	input := make(chan string)
	go func() {
		s := bufio.NewScanner(conn) // 用来接收客户端输入
		for s.Scan() {
			input <- s.Text()
		}
		close(input)
	}()
	// 空闲计时器 (10s) 超时
	timeer := time.NewTimer(10 * time.Second)
	defer timeer.Stop()

	for {
		select {
		case line, ok := <-input:
			if !ok {
				// 客户端断开
				leaving <- clientInfo{ch, who}
				messages <- who + " left"
				conn.Close()
				return
			}
			// 收到消息广播 , 重置计时器
			messages <- who + ": " + line
			timeer.Reset(10 * time.Second)
		case <-timeer.C:
			// 超时空闲 , 离开广播
			fmt.Fprintf(conn, "You are idle for 10s, you are left\n")
			leaving <- clientInfo{ch, who}
			messages <- who + " left"
			conn.Close()
			return
		}
	}
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintf(conn, "%s\n", msg)
	}
}

// broadcaster 广播消息
func broadcaster() {
	// 客户端集合
	clients := make(map[client]string)
	for {
		select {
		case msg := <-messages:
			for cli := range clients {
				// 非阻塞接收 ， 如果clientA 不接受 ， 则跳过
				select {
				case cli <- msg:
					// 接受成功 ， 则发送消息
				default:
					// 接受失败 ， 则跳过
				}
			}
		case ci := <-entering:
			clients[ci.ch] = ci.name
			// 告诉新用户现在有哪些人在线
			var online []string
			for _, name := range clients {
				online = append(online, name)
			}
			ci.ch <- fmt.Sprintf("Now %d users are online: %v\n", len(online), online)

			// 告诉其他人有新用户加入
			for cli := range clients {
				if cli != ci.ch {
					// 非阻塞接收 ， 如果clientA 不接受 ， 则跳过
					select {
					case cli <- fmt.Sprintf("%s joined\n", ci.name):
						// 接受成功 ， 则发送消息
					default:
						// 接受失败 ， 则跳过
					}
				}
			}
		case ci := <-leaving:
			delete(clients, ci.ch)
			close(ci.ch)
			// 告诉其他人有新离开用户
			for cli := range clients {
				if cli != ci.ch {
					// 非阻塞接收 ， 如果clientA 不接受 ， 则跳过
					select {
					case cli <- fmt.Sprintf("%s left\n", ci.name):
						// 接受成功 ， 则发送消息
					default:
						// 接受失败 ， 则跳过
					}
				}
			}
		}
	}
}
