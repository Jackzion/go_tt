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
		io.Copy(os.Stdout, conn)
		log.Println("done")
		done <- struct{}{}
	}()
	mustCopy(conn, os.Stdin)
	// 只关闭写的部分 ， 保留读的部分
	tcpCon := conn.(*net.TCPConn)
	tcpCon.CloseWrite()
	<-done
}

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatal(err)
	}
}
