package parser

import (
	"github.com/emilioastarita/gphp/lexer"
	"github.com/emilioastarita/gphp/node"
)

type Parser struct {
	stream              lexer.TokensStream
	token               lexer.Token
	currentParseContext int
}

func (p *Parser) ParseSourceFile(source string, uri string) {
	p.stream.Source(source)
	p.stream.CreateTokens()
	p.reset()
	sourceFile := node.NewSourceFile(source, uri)
	if p.token.Kind != lexer.EndOfFileToken {
		sourceFile.Add(p.parseInlineHtml(sourceFile))
	}
}

func (p *Parser) reset() {
	p.advanceToken()
	p.currentParseContext = 0
}
func (p *Parser) advanceToken() {
	p.token = p.stream.ScanNext()
}

func (p *Parser) parseInlineHtml(source node.Node) node.Node {
	end := p.eatOptional1(lexer.ScriptSectionEndTag)
	text := p.eatOptional1(lexer.InlineHtml)
	start := p.eatOptional1(lexer.ScriptSectionStartTag)
	node := node.NewInlineHtml(&source, end, text, start)
	return node
}

func (p *Parser) eatOptional1(kind lexer.TokenKind) *lexer.Token {
	t := p.token
	if t.Kind == kind {
		p.advanceToken()
		return &t
	}
	return nil
}
