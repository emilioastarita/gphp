package node

import "github.com/emilioastarita/gphp/lexer"

type errorControlExpression struct {
	unaryOpExpression
}

func NewErrorControlExpression(parent *Node) *errorControlExpression {
	n := &errorControlExpression{}
	n.childNames = [...]string{"operator", "operand"}
	n.parent = parent
	return n
}

func (s *errorControlExpression) Parent() *Node {
	return s.parent
}
