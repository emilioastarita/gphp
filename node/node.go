package node

type node struct {
	parent *Node
}

type Node interface {
	Parent() *Node
}
