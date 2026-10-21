// 本程序演示 geecache 多节点分布式缓存的运行方式。
//
// 使用方法：
//
//	# 1) 分别在 3 个终端启动 3 个缓存节点
//	go run main.go -port=8001
//	go run main.go -port=8002
//	go run main.go -port=8003
//
//	# 2) 启动 API 服务器（-api 模式下仍会同时启动一个缓存节点，详见 main 函数注释）
//	go run main.go -port=8001 -api
//
//	# 3) 通过 API 服务器访问数据（key 会被一致性哈希分配到不同节点）
//	curl "http://localhost:9999/api?key=Tom"
package main

import (
	"flag"
	"fmt"
	"go_tt/geecache"
	"log"
	"net/http"
)

// db 模拟一个慢速数据库（如 MySQL），作为缓存未命中时的"数据源"。
// Getter 回源时会查询这里。
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// createGroup 创建名为 scores 的缓存分组。
// 容量 2<<10（2KB），数据源为上面的 db 模拟数据库。
func createGroup() *geecache.Group {
	// 创建一个 Group：分配独立的 LRU 缓存，并绑定数据源 getter
	return geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		// 数据源：缓存未命中时调用，这里模拟查数据库（带日志，方便观察回源行为）
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// startCacheServer 启动一个缓存节点服务器。
//
// 参数：
//
//	addr  本节点地址，如 http://localhost:8001
//	addrs 集群中所有节点地址（含本节点），用于构建一致性哈希环
//	gee   要加入集群的缓存分组
func startCacheServer(addr string, addrs []string, gee *geecache.Group) {
	// 创建 HTTPPool：既是本节点的 HTTP 服务端，也是选节点的 PeerPicker
	peers := geecache.NewHTTPPool(addr)
	// 将集群所有节点写入一致性哈希环，并为每个远程节点创建 httpGetter
	peers.Set(addrs...)
	// 让 Group 通过 peers 具备"分布式取数"的能力
	gee.RegisterPeers(peers)
	log.Println("geecache is running at", addr)
	// addr 形如 http://localhost:8001，去掉前 7 个字符 "http://" 后交给 HTTP 服务器监听
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// startAPIServer 启动 API 服务器（面向用户的访问入口）。
// 它本身不参与一致性哈希，只负责接收用户请求并调用 gee.Get 获取数据。
// 数据流向：用户 -> API 服务器 -> 本地缓存 / 远程节点 / 数据源
func startAPIServer(apiAddr string, gee *geecache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			// 读取缓存（未命中时 Group 会自动回源或从其他节点取）
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))

}

func main() {
	// 命令行参数：
	//   -port 本进程监听的缓存节点端口（默认 8001）
	//   -api  是否同时启动 API 服务器（默认 false）
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999" // API 服务器固定监听 9999
	// 集群节点拓扑：3 个缓存节点
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	// 提取所有节点地址，组成一致性哈希环所需的地址列表
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup()
	// -api 模式：在后台协程启动 API 服务器（主流程仍会继续启动下面的缓存节点）
	if api {
		go startAPIServer(apiAddr, gee)
	}
	// 启动本进程的缓存节点（阻塞运行）。
	// 注意：addrMap 中只有 8001~8003，若用 -port=9999 -api 启动，
	// addrMap[9999] 会得到空字符串，addr[7:] 将触发 panic（slice bounds out of range）。
	// 因此 -api 模式应搭配 8001~8003 之一使用，让本进程同时充当 API 服务器和一个缓存节点。
	startCacheServer(addrMap[port], []string(addrs), gee)
}
