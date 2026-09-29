package ch8

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

type client chan<- string

var (
	entering = make(chan client) // 管道里传输的是"对讲机"（client类型）
	leaving  = make(chan client) // 管道里传输的是"对讲机"
	messages = make(chan string) // 管道里传输的是"纸条"（字符串）
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
	entering <- ch
	messages <- who + " joined"

	// 读取客户端输入 ， 广播消息
	input := bufio.NewScanner(conn)
	for input.Scan() {
		messages <- who + ": " + input.Text()
	}
	// 离开广播
	leaving <- ch
	messages <- who + " left"
	conn.Close()
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintf(conn, "%s\n", msg)
	}
}

// broadcaster 广播消息
func broadcaster() {
	// 客户端集合
	clients := make(map[client]bool)
	for {
		select {
		case msg := <-messages:
			for cli := range clients {
				cli <- msg
			}
		case cli := <-entering:
			clients[cli] = true
			// 广播"用户X进入"
		case cli := <-leaving:
			clients[cli] = false
			delete(clients, cli)
			// 广播"用户X离开"
			close(cli)
		}
	}
}
