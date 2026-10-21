package geecache

// 本文件实现了基于 HTTP 协议的节点通信，包含两个角色：
//   - HTTPPool：既实现 PeerPicker（客户端视角：为 key 选择目标节点），
//     又实现 http.Handler（服务端视角：接收其他节点发来的数据请求）
//   - httpGetter：实现 PeerGetter，负责向远程节点发送 HTTP 请求
//
// 节点间请求的 URL 格式：
//
//	GET /_geecache/<group>/<key>
//
// 例如：GET /_geecache/scores/Tom

import (
	"fmt"
	"go_tt/geecache/consistenthash"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/" // 所有节点间请求的路径前缀
	defaultReplicas = 50            // 每个真实节点对应的虚拟节点数量
)

// HTTPPool 是节点通信的核心结构，同时扮演两个角色：
//  1. 服务端：作为 http.Handler 接收其他节点发来的数据请求
//  2. 客户端：作为 PeerPicker，通过一致性哈希为 key 选择目标节点
type HTTPPool struct {
	self        string                 // 本节点地址（如 http://localhost:8001），用于判断 key 是否归属本机
	basePath    string                 // 节点间请求的路径前缀
	mu          sync.Mutex             // 互斥锁：保护 peers 和 httpGetters 的并发访问
	peers       *consistenthash.Map    // 一致性哈希环：将 key 映射到节点地址
	httpGetters map[string]*httpGetter // 每个远程节点地址 -> 对应的 httpGetter（按节点复用配置）
}

// httpGetter 是"远程节点客户端"的实现，负责构造 URL 并发起 HTTP GET 请求。
type httpGetter struct {
	baseURL string // 远程节点基础地址 + 路径前缀，如 http://localhost:8002/_geecache/
}

// Get 实现 PeerGetter 接口：向远程节点发送 GET /_geecache/<group>/<key> 请求取值。
// 对 group 和 key 做 URL 转义，防止其中包含 "/" 等特殊字符破坏路径结构。
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	// 发送 GET 请求
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	// 非 200 状态码视为失败（远端可能返回 500 表示其数据源出错等）
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

// 编译期断言：确认 *httpGetter 实现了 PeerGetter 接口。
var _ PeerGetter = (*httpGetter)(nil)

// NewHTTPPool 创建节点通信池，self 为本节点地址。
// 注意：创建后还需调用 Set 将集群节点地址写入哈希环，才能正常选择节点。
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Set 初始化一致性哈希环，并为每个节点创建对应的 httpGetter。
// 通常在启动时调用一次，传入集群中所有节点的地址（包括自己）。
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter)
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{
			baseURL: peer + p.basePath,
		}
	}
}

// PickPeer 实现 PeerPicker：用一致性哈希找到 key 归属的节点。
// 若归属节点是其他机器，返回其 httpGetter，由调用方向其发起远程请求；
// 若归属节点就是本机（或哈希环为空），返回 ok=false，调用方走本地加载。
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

// 编译期断言：确认 *HTTPPool 实现了 PeerPicker 接口。
var _ PeerPicker = (*HTTPPool)(nil)

// Log 输出带节点标识的日志，方便区分多节点同时运行时的输出来源。
func (p *HTTPPool) Log(format string, args ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, args...))
}

// ServeHTTP 实现 http.Handler：处理其他节点发来的数据请求（服务端视角）。
//
// 处理流程：
//  1. 校验路径前缀 /_geecache/
//  2. 从路径中解析出 group 和 key
//  3. 通过 GetGroup 找到分组，调用其 Get 方法（未命中会自动触发回源）
//  4. 将结果写入响应体
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		http.NotFound(w, r)
		return
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	// 期望的路径格式：/<basePath>/<group>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	groupname := parts[0]
	key := parts[1]

	// 根据分组名取出 Group（分组在进程启动时通过 NewGroup 注册到全局表）
	group := GetGroup(groupname)
	if group == nil {
		http.Error(w, "group not found", http.StatusBadRequest)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}
