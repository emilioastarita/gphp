package node

import "github.com/emilioastarita/gphp/lexer"

type inlineHtml struct {
	node
	scriptSectionEndTag   *lexer.Token
	text                  *lexer.Token
	scriptSectionStartTag *lexer.Token
	childNames            [3]string
}

func NewInlineHtml(parent *Node, endTag *lexer.Token, text *lexer.Token, startTag *lexer.Token) *inlineHtml {
	n := &inlineHtml{
		scriptSectionEndTag: endTag,
		text:                text,
		scriptSectionStartTag: startTag,
		childNames:            [3]string{"scriptSectionEndTag", "text", "scriptSectionStartTag"},
	}
	n.parent = parent
	return n
}

func (s *inlineHtml) Parent() *Node {
	return s.parent
}
