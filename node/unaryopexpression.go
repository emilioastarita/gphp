package node

import "github.com/emilioastarita/gphp/lexer"

type unaryOpExpression struct {
	node
	Operator   *lexer.Token
	Operand    *Node
	childNames [2]string
}

func NewUnaryOpExpression(parent *Node) *unaryOpExpression {
	n := &unaryOpExpression{
		childNames: [...]string{
			"operator",
			"operand"},
	}
	n.parent = parent
	return n
}

func (s *unaryOpExpression) Parent() *Node {
	return s.parent
}
