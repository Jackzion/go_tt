package ch8

import (
	"io"
	"log"
	"net"
	"os"
)

func TestNetcat1() {
	conn, err := net.Dial("tcp", "localhost:8999")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	done := make(chan struct{})
	go func() {
		io.Copy(os.Stdout, conn) // 服务器关闭后，这里会返回
		log.Println("done")
		done <- struct{}{} // 通知主线程
	}()
	mustCopy(conn, os.Stdin) // 读键盘输入，写到服务器

	tcpCon := conn.(*net.TCPConn)
	tcpCon.CloseWrite() // 关闭写端
	<-done              // 等 goroutine 读完
}

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatal(err)
	}
}
