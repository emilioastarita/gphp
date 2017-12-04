package node

import "github.com/emilioastarita/gphp/lexer"

type ifStatement struct {
	node
	IfKeyword     *lexer.Token
	OpenParen     *lexer.Token
	Expression    *Node
	CloseParen    *lexer.Token
	Colon         *lexer.Token
	Statements    []*Node
	ElseIfClauses []*Node
	ElseClause    *Node
	EndifKeyword  *lexer.Token
	SemiColon     *lexer.Token
	childNames    [10]string
}

func NewIfStatement(parent *Node) *ifStatement {
	n := &ifStatement{
		childNames: [...]string{
			"ifKeyword",
			"openParen",
			"expression",
			"closeParen",
			"colon",
			"statements",
			"elseIfClauses",
			"elseClause",
			"endifKeyword",
			"semicolon"},
	}
	n.parent = parent
	return n
}

func (s *ifStatement) Parent() *Node {
	return s.parent
}
