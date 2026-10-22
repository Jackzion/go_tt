// Package geecache 实现了一个与 groupcache 类似的分布式缓存系统。
//
// 核心概念：
//
//	Group（缓存分组）：缓存的命名空间，每个 Group 拥有独立的缓存和独立的数据源（Getter）。
//	    例如可以为 "scores"、"students" 各建一个 Group，互不干扰。
//	Getter（数据源）：定义缓存未命中时如何从源头（数据库、文件、RPC 等）加载数据。
//	PeerPicker / PeerGetter（节点通信）：分布式模式下，负责把 key 路由到归属节点并取回数据。
//
// 一次读取（Group.Get）的完整流程：
//
//  1. 查本地缓存（mainCache），命中则直接返回
//  2. 未命中 -> 通过 PeerPicker（一致性哈希）计算 key 归属的节点
//     - 归属远程节点 -> 通过 PeerGetter 发起远程请求取值
//     - 归属本机     -> 调用本地数据源 getter.Get 加载（如查数据库）
//  3. 加载到的数据会回填到本地缓存，下次即可直接命中
package geecache

import (
	"errors"
	pb "go_tt/geecache/geecachepb"
	"go_tt/geecache/singleflight"
	"log"
	"sync"
)

// Getter 定义了从数据源（如数据库、文件、RPC 等）加载数据的接口。
// 缓存未命中且 key 归属本机时，会调用该接口完成"回源"。
type Getter interface {
	// Get 返回 key 对应的字节数据；key 不存在时应返回 error。
	Get(key string) ([]byte, error)
}

// GetterFunc 是一个函数类型，通过类型转换即可适配 Getter 接口，
// 免去为简单逻辑单独定义结构体的麻烦（函数式适配器模式）。
//
// 用法：
//
//	geecache.GetterFunc(func(key string) ([]byte, error) {
//	    // 从数据库等数据源取值
//	})
type GetterFunc func(key string) ([]byte, error)

// Get 实现 Getter 接口：直接调用被转换的函数 f。
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group 代表一个命名的缓存分组，是用户使用 geecache 的入口。
type Group struct {
	name      string     // 分组名称，如 "scores"；远程节点间通过它识别同一批数据
	getter    Getter     // 数据源：缓存未命中时的回源逻辑；不同 Group 可挂不同的数据源
	mainCache cache      // 本节点的主缓存（并发安全的 LRU）
	peers     PeerPicker // 节点选择器：计算 key 应去哪个节点取数；为 nil 表示单机模式
	// singleflight.Group 用于管理不同 key 的请求(call) - waitGroup + mulock 实现一个 key 只会触发一次 fn 请求
	singleflightGroup *singleflight.Group
}

// RegisterPeers 注册节点选择器，使本 Group 具备分布式能力。
// 只能注册一次：底层 HTTPPool 同时充当服务端与客户端调度器，重复注册通常说明配置有误。
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// load 在本地缓存未命中时加载数据，是分布式逻辑的核心。
// 优先从远程节点取（一致性哈希算出的归属节点），取不到再回退到本地数据源。
func (g *Group) load(key string) (value ByteView, err error) {
	// 用 singleflight.Group 确保每个 key 只会触发一次 fn 请求 ， 避免重复请求
	view, err := g.singleflightGroup.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			// 第一优先级：查询该 key 归属的远程节点
			if peer, ok := g.peers.PickPeer(key); ok {
				// 从远程节点获取值
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				// 远程取值失败（网络异常、远端数据源出错等），记录日志后走本地兜底
				log.Println("[GeeCache] Failed to get from peer", err)
			}
			// PickPeer 返回 false：该 key 归属本机，直接走下方的本地加载
		}
		// 第二优先级（兜底）：本地数据源（getter）
		return g.getLocally(key)
	})
	if err == nil {
		value = view.(ByteView)
		return value, nil
	}
	return ByteView{}, err
}

// getFromPeer 通过 PeerGetter 从远程节点获取数据，并包装为 ByteView。
func (g *Group) getFromPeer(peer PeerGetter, key string) (value ByteView, err error) {
	// 使用 protobuf 定义的 Request 和 Response 结构体
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	resp := &pb.Response{}
	err = peer.Get(req, resp)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{resp.Value}, nil
}

var (
	mu     sync.RWMutex              // 读写锁：保护全局分组注册表
	groups = make(map[string]*Group) // 全局分组注册表：分组名 -> *Group
)

// NewGroup 创建一个缓存分组并注册到全局表中。
//
// 参数：
//
//	name       分组名称，必须唯一
//	cacheBytes 该分组的缓存容量上限（字节）
//	getter     数据源，不能为空（否则缓存永远无法填充数据）
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("getter is nil")
	}
	mu.Lock()
	defer mu.Unlock()
	groups[name] = &Group{
		name:   name,
		getter: getter,
		mainCache: cache{
			cacheBytes: cacheBytes,
		},
		singleflightGroup: &singleflight.Group{},
	}
	return groups[name]
}

// GetGroup 根据名称获取已注册的分组；未注册时返回 nil。
// 远程节点收到节点间 HTTP 请求后，就是用它找到对应分组来取数据的。
func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	return groups[name]
}

// Get 是用户代码（如 API 服务器）读取缓存的主入口。
// 先查本地缓存，未命中则走 load 流程（远程节点 -> 本地数据源）。
func (g *Group) Get(key string) (value ByteView, err error) {
	if key == "" {
		return ByteView{}, errors.New("empty key")
	}
	if value, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return value, nil
	}
	// 本地缓存未命中，走分布式加载流程
	return g.load(key)
}

// getLocally 调用本地数据源（getter）加载数据。
// 加载成功后会通过 populateCache 回填到本地缓存，下次即可直接命中。
func (g *Group) getLocally(key string) (ByteView, error) {
	b, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	// 克隆字节片：防止数据源后续修改原始数据，保证缓存值不可变
	value := ByteView{cloneBytes(b)}
	g.populateCache(key, value)
	return value, nil
}

// populateCache 将数据写入本地主缓存。
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
