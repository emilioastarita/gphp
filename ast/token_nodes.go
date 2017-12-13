package ast

import (
	"github.com/emilioastarita/gphp/lexer"
)

type Missing struct {
	CNode `serialize:"-"`
	Token *lexer.Token
}

type SkippedNode struct {
	CNode `serialize:"-"`
	Token *lexer.Token
}

type TokenNode struct {
	CNode `serialize:"-"`
	Token *lexer.Token
}

func (n *SkippedNode) SetToken(t *lexer.Token) {
	n.Token = t
}

func (n *SkippedNode) GetToken() *lexer.Token {
	return n.Token
}

func (n *TokenNode) SetToken(t *lexer.Token) {
	n.Token = t
}

func (n *TokenNode) GetToken() *lexer.Token {
	return n.Token
}

func (n *Missing) SetToken(t *lexer.Token) {
	n.Token = t
}

func (n *Missing) GetToken() *lexer.Token {
	return n.Token
}

func NewSkippedNode(from *lexer.Token) *SkippedNode {
	t := &lexer.Token{Kind: from.Kind, FullStart: from.FullStart, Start: from.Start, Length: from.Length, Missing: true}
	skipped := &SkippedNode{}
	skipped.Token = t
	return skipped
}
