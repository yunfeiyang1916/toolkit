package gin

// http方法树
type methodTree struct {
	method string
	root   *node
}

// http方法树数组
type methodTrees []methodTree

type nodeType uint8

const (
	// 根节点
	root = iota + 1
	// 参数路由，比如 /user/:id
	param
	// 匹配所有内容的路由，比如 /article/*key
	catchAll
)

// 前缀树(字典树)节点
// 如果我们有两个路由，分别是 /index，/inter，
// 则根节点为 {path: "/in", indices: "dt"}
// 两个子节点为{path: "dex", indices: ""}，{path: "ter", indices: ""}
type node struct {
	// 当前节点相对路径（与祖先节点的 path 拼接可得到完整路径）
	path string
	// 所有孩子节点的path[0]组成的字符串
	indices string
	// 孩子节点是否有通配符（wildcard）
	wildChild bool
	// 节点类型
	nType nodeType
	// 当前节点及子孙节点的实际路由数量
	priority uint32
	// 孩子节点
	children []*node
	handlers HandlersChain
	fullPath string
}
