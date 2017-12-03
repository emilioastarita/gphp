package node

import "github.com/emilioastarita/gphp/lexer"

type compoundStatement struct {
	node
	OpenBrace  *lexer.Token
	Statements []*Node
	CloseBrace *lexer.Token
	childNames [3]string
}

func NewCompoundStatement(parent *Node) *compoundStatement {
	n := &compoundStatement{
		childNames: [3]string{"openBrace", "statements", "closeBrace"},
	}
	n.parent = parent
	return n
}

func (s *compoundStatement) Parent() *Node {
	return s.parent
}
