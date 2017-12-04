package node

import "github.com/emilioastarita/gphp/lexer"

type missing struct {
	node
	Token *lexer.Token
}

func NewMissing(parent *Node) *missing {
	n := &missing{}
	n.parent = parent
	return n
}

func (s *missing) Parent() *Node {
	return s.parent
}
