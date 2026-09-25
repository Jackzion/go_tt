package ch8

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func FtpServer() {
	linstener, err := net.Listen("tcp", "localhost:2121")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("ftp server linsten on localhost:2121")
	for {
		conn, err := linstener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handleFtp(conn)
	}
}

func handleFtp(conn net.Conn) {
	defer conn.Close()
	fmt.Fprintf(conn, "welcome to go FTP")

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		args := strings.Fields(line) // 把 "ls -l" 拆成 ["ls", "-l"]
		if len(args) == 0 {
			continue
		}

		cmd := args[0]
		switch cmd {
		case "ls":
			// 读取当前目录的文件列表
			files, err := os.ReadDir("./")
			if err != nil {
				fmt.Fprintf(conn, "Error: %v\n", err)
				continue
			}
			// 遍历文件并发送给客户端
			for _, file := range files {
				info, err := file.Info()
				if err != nil {
					fmt.Fprintf(conn, "Error: %v\n", err)
					continue
				}
				// 格式化输出：文件名 ， 大小 ，修改时间
				fmt.Fprintf(conn, "%-20s %8d bytes  %s\n",
					file.Name(),
					info.Size(),
					info.ModTime().Format("2006-01-02 15:04:05"))
			}
		case "cd":
			if len(args) != 2 {
				fmt.Fprintf(conn, "Usage: cd <dirname>\n")
				continue
			}
			dir := args[1]
			err := os.Chdir(dir)
			if err != nil {
				fmt.Fprintf(conn, "Error: %v\n", err)
				continue
			} else {
				// 获取当前目录并发送给客户端
				cwd, err := os.Getwd()
				if err != nil {
					fmt.Fprintf(conn, "Error: %v\n", err)
					continue
				}
				fmt.Fprintf(conn, "Changed to %s\n", cwd)
			}
		case "get":
			if len(args) != 2 {
				fmt.Fprintf(conn, "Usage: get <filename>\n")
				continue
			}
			filename := args[1]
			// 读取文件内容
			content, err := os.ReadFile(filename)
			if err != nil {
				fmt.Fprintf(conn, "Error: %v\n", err)
				continue
			}
			// 发送文件内容给客户端
			conn.Write(content)
		case "send":
			if len(args) != 2 {
				fmt.Fprintf(conn, "Usage: send <filename>\n")
				continue
			}
			filename := args[1]
			// 告诉客户端可以发送文件了
			fmt.Fprintf(conn, "Ready to receive file %s\n", filename)

			// 读取 client 发送的文件内容
			var content []byte
			for scanner.Scan() {
				line = scanner.Text()
				if line == "END" {
					break
				}
				content = append(content, []byte(line+"\n")...)
			}
			// 保存文件
			err := os.WriteFile(filename, content, 0644)
			if err != nil {
				fmt.Fprintf(conn, "Error: %v\n", err)
				continue
			} else {
				fmt.Fprintf(conn, "File %s saved successfully\n", filename)

			}
		case "close":
			fmt.Fprintf(conn, "Bye!\n")
			return
		default:
			fmt.Fprintf(conn, "Unknown command: %s\n", cmd)
		}

	}
}
