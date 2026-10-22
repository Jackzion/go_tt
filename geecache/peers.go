package geecache

import pb "go_tt/geecache/geecachepb"

// 本文件定义了节点间交互的两个核心接口：
//   - PeerPicker 负责"选节点"：根据 key 算出数据归属哪个节点
//   - PeerGetter 负责"从节点取数据"：向具体节点发起请求并读取结果
//
// 分离这两个职责后，更换通信方式（例如把 HTTP 换成 gRPC）
// 只需提供新的实现，不必改动 Group 的核心逻辑。

// PeerPicker 根据 key 选择一个合适的节点来获取数据。
//
// 由 geecache.HTTPPool 实现：内部使用一致性哈希把 key 映射到集群中的某个节点；
// 如果该节点恰好是本机，则返回 ok=false，
// 表示应该直接从本地缓存/本地数据源获取，无需远程调用。
type PeerPicker interface {
	// PickPeer 选出 key 所属的节点。
	// ok=false 表示该 key 无需远程访问（通常是因为归属节点就是本机）。
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 定义了"从某个具体节点上读取数据"的方式。
//
// 由 geecache.httpGetter 实现：向远程节点发送
//
//	GET /_geecache/<group>/<key>
//
// 请求，读取响应体作为缓存值。
type PeerGetter interface {
	// Get 从指定分组（group）中读取指定 key 的值。
	// group 用于区分多个缓存分组，避免跨分组取错数据。
	// 因为分布式节点都共用同一套代码，因此使用 proto 序列化会变得非常轻松，因为大家都能拿到密本 pb
	Get(in *pb.Request, out *pb.Response) error
}
