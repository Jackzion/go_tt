package geecache

// PeerPicker 选择一个节点来获取数据
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 从节点获取获取数据
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}
