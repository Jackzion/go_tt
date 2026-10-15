package geecache

import (
	"errors"
	"log"
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 是 Getter 接口的实现，它将函数转换为 Getter 接口类型。
type GetterFunc func(key string) ([]byte, error)

// Get 实现 Getter 接口，调用函数 f 来获取值。
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

type Group struct {
	name      string // Group 名称
	getter    Getter // 获取器 - 不同 Group 可以有不同的获取器,比如从数据库、缓存、local等获取
	mainCache cache  // 主缓存 - 用于存储 Group 内的键值对,缓存大小由 cacheBytes 控制
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup 创建一个新的 Group
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
	}
	return groups[name]
}

// GetGroup 获取 Group
func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	return groups[name]
}

// Get 获取 Group 内的值
func (g *Group) Get(key string) (value ByteView, err error) {
	if key == "" {
		return ByteView{}, errors.New("empty key")
	}
	if value, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return value, nil
	}
	return g.load(key)
}

// load 从获取器中获取值
func (g *Group) load(key string) (value ByteView, err error) {
	return g.getLocally(key)
}

// getLocally 从本地中获取值
func (g *Group) getLocally(key string) (ByteView, error) {
	b, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	// 克隆字节片，避免修改原始数据
	value := ByteView{cloneBytes(b)}
	g.populateCache(key, value)
	return value, nil
}

// populateCache 填充主缓存
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
