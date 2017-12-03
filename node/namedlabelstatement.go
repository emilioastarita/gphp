package node

import "github.com/emilioastarita/gphp/lexer"

type namedLabelStatement struct {
	node
	Name       *lexer.Token
	Colon      *lexer.Token
	Statement  *Node
	childNames [3]string
}

func NewNamedLabelStatement(parent *Node) *namedLabelStatement {
	n := &namedLabelStatement{
		childNames: [3]string{"name", "colon", "statement"},
	}
	n.parent = parent
	return n
}

func (s *namedLabelStatement) Parent() *Node {
	return s.parent
}
