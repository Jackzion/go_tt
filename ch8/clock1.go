package ch8

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

func TestClock1() {
	// 增加一个命令行参数，用于指定服务端监听的端口号
	port := flag.Int("port", 8999, "TCP port to listen on")
	flag.Parse() // parse the command-line flags
	// 服务端暴露端口
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err) // e.g., connection aborted
			continue
		}
		// 每次接收到连接请求时，启动一个新的 goroutine 来处理该连接 , 以支持多并发
		go handleConn(conn) // handle one connection at a time
	}

}

func handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		_, err := io.WriteString(conn, time.Now().Format("15:04:05\n"))
		if err != nil {
			return // e.g., client disconnected
		}
		time.Sleep(1 * time.Second)
	}
}
