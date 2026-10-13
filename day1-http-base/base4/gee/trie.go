package gee

import (
	"strings"
)

// ===========================================================================
// 本文件是动态路由的心脏：前缀树（Trie）。
//
// 【它解决什么问题？】
// 之前路由存在 map 里，只能精确匹配 "/hello"。
// 现在要支持 "/hello/:name"（: 匹配任意一段）和 "/assets/*filepath"（* 吞掉剩余所有段）。
// map 做不了这件事，于是把路由"按 / 切段"后挂成一棵树。
//
// 【注册两条路由后树长什么样？】
//
//	root                                   每个方框是一个 node 结构体
//	 └── p                pattern=""       ← 中转站：路由还没走完，没证书
//	      ├── :lang       pattern=""  isWild=true
//	      │    └── doc    pattern="/p/:lang/doc" ★ 终点：有证书！
//	      └── *filepath   pattern="/p/*filepath" ★ isWild=true 证书
//
//	注册的路由：/p/:lang/doc 和 /p/*filepath
//	注意 "p" 节点被两条路由共享 —— 这就是"前缀复用"。
//
// 【pattern 什么时候有值？】
// 只在"一条路由的最后一段"对应的节点上。中转站节点一律为空串。
// 所以判断匹配是否成功，唯一的金标准是：走到终点 && n.pattern != ""。
// 例如请求 /p/python：能走到 :lang（通配匹住了 "python"），
// 但 :lang.pattern == "" —— 它只是路过，这条路由并不存在 → 404。
//
// 【两个核心动作】
//   - insert：启动时注册路由（把每一段挂上树，终点贴证书）
//   - search：每个请求到来时拿着路径逐段下探，找挂着证书的节点
// ===========================================================================

// node 是前缀树的一个节点，代表路径中的"一段"。
type node struct {
	pattern  string  // 完整路由规则（如 /p/:lang/doc）。中转站为空串，终点才有值
	part     string  // 本节点代表的路径段（如 ":lang"）
	children []*node // 子节点切片（同一层可能有多个候选：精确的、:通配的、*通配的）
	isWild   bool    // part 以 ':' 或 '*' 开头 → 匹配时不看内容，什么都能匹住
}

// matchChild 在子节点里找【第一个】能接住 part 的节点。
// 给 insert 用：插入是确定性的，找到落脚点就开工，不需要候选列表。
// 注意副作用：child.isWild 为真就直接返回，不区分精确命中还是通配将就
// —— 这会导致"先注册的 :lang 吞掉后来的 go 静态段"（原教程的已知局限）。
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

// matchChildren 收集【所有】能接住 part 的子节点，返回候选列表。
// 给 search 用：查询可能要试错——比如 "go" 同时能被 :lang（通配）和 go（精确）匹住，
// :lang 那条路往下走不通时，必须回头换 go 那条路重试（回溯）。
// 所以要"全部候选"而不是"第一个"。
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

// insert 把一条路由规则挂到树上。可以理解为"带着爬楼进度表 height 递归放箱子"：
//
//	契约：站在 n 节点，把 parts[height:] 这些剩余段插进 n 的子树。
//
// 以 insert(root, "/p/:lang/doc", ["p",":lang","doc"], 0) 为例，逐层推演：
//
//	height=0 拿着 "p"    → 没有孩子能接 → 新建 p 节点   → 交给它继续爬(height=1)
//	height=1 拿着 ":lang" → 没有孩子能接 → 新建 :lang   → 交给它继续爬(height=2)
//	height=2 拿着 "doc"  → 没有孩子能接 → 新建 doc     → 交给它继续爬(height=3)
//	height=3 段数用完了   → 到站！给 doc 贴证书 pattern="/p/:lang/doc"，逐层返回
//
// 三个要点：
//  1. height 是"爬到第几段了"的游标，比每层切片 parts[height+1:] 省内存
//  2. 发证（n.pattern = pattern）只发生在到站的那一刻，所以中转站的 pattern 永远是空
//  3. child.insert 写在 if 外面：无论 child 是现成的还是新建的，都必须递归进去
//     —— 新建的孩子肚子里还有没放完的段呢
func (n *node) insert(pattern string, parts []string, height int) {
	// 到站条件：手里没有剩余段了，当前节点就是这条路由的终点 → 发证书
	if len(parts) == height {
		n.pattern = pattern
		return
	}

	// 还没到站：看看手里这段是什么
	part := parts[height]

	// 找一个能接住这段的孩子；没有就现建一个（: 或 * 开头则标记为通配）
	child := n.matchChild(part)
	if child == nil {
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}

	// 把"剩下的段"交给这个孩子，让它继续爬下一层
	child.insert(pattern, parts, height+1)
}

// search 拿请求路径在树里逐段下探，找到挂证书的节点；找不到返回 nil。
//
//	契约：在 n 的子树里查找 parts[height:]，命中返回终点节点，否则 nil。
//
// 退出条件有两个，含义完全不同：
//   - len(parts) == height：请求路径走完了，看看这个节点有没有证书（有→命中，无→404）
//   - n.part 以 "*" 开头：遇到贪婪捕获符，剩余段全归它管，不许再往下走
//     （区分 * 和 : 的关键：两者都 isWild，但 : 只匹一段必须继续走，* 一口吞就地结束）
//
// 为什么要 matchChildren 返回候选列表逐个试？
// 因为一个段可能被多个孩子匹住（精确的 go 和通配的 :lang），
// 第一条路走不通时要回头试下一条 —— 这就是循环里的回溯：
//
//	for _, child := range children {
//	    result := child.search(parts, height+1)
//	    if result != nil { return result }  // 某条路通了，立刻返回
//	    // 没人返回成功 → 自然进入下一轮，换条路重试
//	}
func (n *node) search(parts []string, height int) *node {
	// 退出条件一：路径走完，或遇到 * 就地收官
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil // 能走到这但没证书 = 只是路过（如 /p/python 走到了 :lang）
		}
		return n
	}

	// 还没到终点：收集本层所有能匹住当前段的孩子（可能 0 个、1 个或多个）
	part := parts[height]
	children := n.matchChildren(part)

	// 逐个候选尝试；任何一条路找到证书就成功返回；全部死路才认输
	for _, child := range children {
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}

	return nil
}

// parsePath 把 URL 按 "/" 切成段，并丢掉空串：
//
//	"/p//go/"  →  ["p", "go"]
//	"/"        →  []           （空路径：切出来全是空段，丢完就剩空切片）
//
// 丢空段很关键："/hello/" 和 "/hello" 应该是同一条路由，不能因为多个斜杠就算不同的路。
func parsePath(pattern string) []string {
	vs := strings.Split(pattern, "/")
	parts := make([]string, 0, len(vs))
	for _, v := range vs {
		if v != "" {
			parts = append(parts, v)
		}
	}
	return parts
}


