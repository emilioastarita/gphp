package parser

import (
	"github.com/emilioastarita/gphp/lexer"
	"github.com/emilioastarita/gphp/node"
)

type Parser struct {
	stream              lexer.TokensStream
	token               lexer.Token
	currentParseContext ParseContext
}

type ParseContext uint

const (
	SourceElements = iota
	BlockStatements
	ClassMembers
	IfClause2Elements
	SwitchStatementElements
	CaseStatementElements
	WhileStatementElements
	ForStatementElements
	ForeachStatementElements
	DeclareStatementElements
	InterfaceMembers
	TraitMembers
	Count
)

func (p *Parser) ParseSourceFile(source string, uri string) {
	p.stream.Source(source)
	p.stream.CreateTokens()
	p.reset()
	sourceFile := node.NewSourceFile(source, uri)
	if p.token.Kind != lexer.EndOfFileToken {
		sourceFile.Add(p.parseInlineHtml(sourceFile))
	}
	list := p.parseList(sourceFile, SourceElements)
	sourceFile.Merge(list)
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
	n := node.NewInlineHtml(&source, end, text, start)
	return n
}

func (p *Parser) eatOptional1(kind lexer.TokenKind) *lexer.Token {
	t := p.token
	if t.Kind == kind {
		p.advanceToken()
		return &t
	}
	return nil
}

func (p *Parser) parseList(parentNode node.Node, context ParseContext) []*node.Node {
	savedCurrentParseContext := p.currentParseContext
	p.currentParseContext |= 1 << context
	parseListElementFn := p.getParseListElementFn(context)
}

func (p *Parser) getParseListElementFn(context ParseContext) func(*node.Node) node.Node {
	switch context {
	case SourceElements,
		BlockStatements,
		IfClause2Elements,
		CaseStatementElements,
		WhileStatementElements,
		ForStatementElements,
		ForeachStatementElements,
		DeclareStatementElements:
		return p.parseStatementFn()
	case ClassMembers:
		return p.parseClassElementFn()

	case TraitMembers:
		return p.parseTraitElementFn()

	case InterfaceMembers:
		return p.parseInterfaceElementFn()

	case SwitchStatementElements:
		return p.parseCaseOrDefaultStatement()
	default:
		panic("Unrecognized parse context")
	}
}

func (p *Parser) parseStatementFn() func(*node.Node) node.Node {
	return func(parentNode *node.Node) node.Node {
		token := p.token
		switch token.Kind {
		// compound-statement
		case lexer.OpenBraceToken:
			return p.parseCompoundStatement(parentNode)

			// labeled-statement
		case lexer.Name:
			if p.lookahead(lexer.ColonToken) {
				return p.parseNamedLabelStatement(parentNode)
			}
			break

			// selection-statement
		case lexer.IfKeyword:
			return p.parseIfStatement(parentNode)
		case lexer.SwitchKeyword:
			return p.parseSwitchStatement(parentNode)

			// iteration-statement
		case lexer.WhileKeyword: // while-statement
			return p.parseWhileStatement(parentNode)
		case lexer.DoKeyword: // do-statement
			return p.parseDoStatement(parentNode)
		case lexer.ForKeyword: // for-statement
			return p.parseForStatement(parentNode)
		case lexer.ForeachKeyword: // foreach-statement
			return p.parseForeachStatement(parentNode)

			// jump-statement
		case lexer.GotoKeyword: // goto-statement
			return p.parseGotoStatement(parentNode)
		case lexer.ContinueKeyword: // continue-statement
		case lexer.BreakKeyword: // break-statement
			return p.parseBreakOrContinueStatement(parentNode)
		case lexer.ReturnKeyword: // return-statement
			return p.parseReturnStatement(parentNode)
		case lexer.ThrowKeyword: // throw-statement
			return p.parseThrowStatement(parentNode)

			// try-statement
		case lexer.TryKeyword:
			return p.parseTryStatement(parentNode)

			// declare-statement
		case lexer.DeclareKeyword:
			return p.parseDeclareStatement(parentNode)

			// function-declaration
		case lexer.FunctionKeyword:
			// Check that this is not an anonymous-function-creation-expression
			if p.lookahead(p.nameOrKeywordOrReservedWordTokens) || p.lookahead(lexer.AmpersandToken, p.nameOrKeywordOrReservedWordTokens) {
				return p.parseFunctionDeclaration(parentNode)
			}
			break

			// class-declaration
		case lexer.FinalKeyword:
		case lexer.AbstractKeyword:
			if !p.lookahead(lexer.ClassKeyword) {
				p.advanceToken()
				return node.newSkippedToken(token)
			}
		case lexer.ClassKeyword:
			return p.parseClassDeclaration(parentNode)

			// interface-declaration
		case lexer.InterfaceKeyword:
			return p.parseInterfaceDeclaration(parentNode)

			// namespace-definition
		case lexer.NamespaceKeyword:
			if !p.lookahead(lexer.BackslashToken) {
				// TODO add error handling for the case where a namespace definition does not occur in the outer-most scope
				return p.parseNamespaceDefinition(parentNode)
			}

			// namespace-use-declaration
		case lexer.UseKeyword:
			return p.parseNamespaceUseDeclaration(parentNode)

		case lexer.SemicolonToken:
			return p.parseEmptyStatement(parentNode)

			// trait-declaration
		case lexer.TraitKeyword:
			return p.parseTraitDeclaration(parentNode)

			// global-declaration
		case lexer.GlobalKeyword:
			return p.parseGlobalDeclaration(parentNode)

			// const-declaration
		case lexer.ConstKeyword:
			return p.parseConstDeclaration(parentNode)

			// function-static-declaration
		case lexer.StaticKeyword:
			// Check that this is not an anonymous-function-creation-expression
			if !p.lookahead([]lexer.TokenKind{lexer.FunctionKeyword, lexer.OpenParenToken, lexer.ColonColonToken}) {
				return p.parseFunctionStaticDeclaration(parentNode)
			}
		case lexer.ScriptSectionEndTag:
			return p.parseInlineHtml(parentNode)
		}

		//expressionStatement = new ExpressionStatement();
		//expressionStatement.parent = parentNode;
		//expressionStatement.expression = p.parseExpression(expressionStatement, true);
		//expressionStatement.semicolon = p.eatSemicolonOrAbortStatement();
		//return expressionStatement;
	}
}

func (p *Parser) parseStatement(parentNode node.Node) *node.Node {
	fn := p.parseStatementFn()
	st := fn(&parentNode)
	return &st
}

func (p *Parser) parseIfStatement(parentNode *node.Node) node.Node {

}

func (p *Parser) parseNamedLabelStatement(parentNode *node.Node) node.Node {
	st := node.NewNamedLabelStatement(parentNode)
	st.Name = p.eat1(lexer.Name)
	st.Colon = p.eat1(lexer.ColonToken)
	st.Statement = p.parseStatement(st)
	return st
}

func (p *Parser) parseCompoundStatement(parentNode *node.Node) node.Node {
	st := node.NewCompoundStatement(parentNode)
	st.OpenBrace = p.eat1(lexer.OpenBraceToken)
	st.Statements = p.parseList(st, BlockStatements)
	st.CloseBrace = p.eat1(lexer.CloseBraceToken)
	return st
}

func (p *Parser) lookahead(expectedKinds ...lexer.TokenKind) bool {
	startPos := p.stream.Pos
	startToken := p.token
	succeeded := true
	for _, kind := range expectedKinds {
		token := p.stream.ScanNext()
		currentPos := p.stream.Pos
		eofPos := p.stream.EofPos
		if currentPos > eofPos || token.Kind != kind {
			succeeded = false
			break
		}
	}
	p.stream.Pos = startPos
	p.token = startToken
	return succeeded
}

func (p *Parser) eat1(kind lexer.TokenKind) *lexer.Token {
	token := p.token
	if token.Kind == kind {
		p.advanceToken()
		return &token
	}
	return &lexer.Token{kind, token.FullStart, token.FullStart, 0}
}
