package parser

import (
	"lexer"
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

}

func (p *Parser) reset() {
	p.advanceToken()
	p.currentParseContext = 0
}
func (p *Parser) advanceToken() {
	p.token = p.stream.ScanNext()
}
