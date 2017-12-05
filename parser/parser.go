package parser

import (
	"github.com/emilioastarita/gphp/lexer"
	"github.com/emilioastarita/gphp/ast"
)

type Parser struct {
	stream              lexer.TokensStream
	token               lexer.Token
	currentParseContext ParseContext
}

type ParseContext uint

const (
	SourceElements           = iota
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
	sourceFile := ast.SourceFile{P:nil, FileContents: source, Uri: uri}
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

func (p *Parser) parseInlineHtml(source ast.Node) ast.Node {
	end := p.eatOptional1(lexer.ScriptSectionEndTag)
	text := p.eatOptional1(lexer.InlineHtml)
	start := p.eatOptional1(lexer.ScriptSectionStartTag)
	n := ast.InlineHtml{&source, end, text, start}
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

func (p *Parser) parseList(parentNode ast.Node, context ParseContext) []*ast.Node {
	savedCurrentParseContext := p.currentParseContext
	p.currentParseContext |= 1 << context
	parseListElementFn := p.getParseListElementFn(context)
}

func (p *Parser) parseExpressionFn() func(*ast.Node) ast.Node {
	return func(parent *ast.Node) ast.Node {
		return p.parseBinaryExpressionOrHigher(0, parent)
	}
}

func (p *Parser) parseUnaryExpressionOrHigher(parentNode ast.Node) ast.Node {
	token := p.token
	switch token.Kind {
	case lexer.PlusToken,
		lexer.MinusToken,
		lexer.ExclamationToken,
		lexer.TildeToken:
		return p.parseUnaryOpExpression(parentNode);

		// error-control-expression
	case lexer.AtSymbolToken:
		return p.parseErrorControlExpression(parentNode);

		// prefix-increment-expression
	case lexer.PlusPlusToken,
		// prefix-decrement-expression
		lexer.MinusMinusToken:
		return p.parsePrefixUpdateExpression(parentNode);

	case lexer.ArrayCastToken,
		lexer.BoolCastToken,
		lexer.DoubleCastToken,
		lexer.IntCastToken,
		lexer.ObjectCastToken,
		lexer.StringCastToken,
		lexer.UnsetCastToken:
		return p.parseCastExpression(parentNode);

	case lexer.OpenParenToken:
		// TODO remove duplication
		if (p.lookahead(
			[]lexer.TokenKind{lexer.ArrayKeyword,
				lexer.BinaryReservedWord,
				lexer.BoolReservedWord,
				lexer.BooleanReservedWord,
				lexer.DoubleReservedWord,
				lexer.IntReservedWord,
				lexer.IntegerReservedWord,
				lexer.FloatReservedWord,
				lexer.ObjectReservedWord,
				lexer.RealReservedWord,
				lexer.StringReservedWord,
				lexer.UnsetKeyword}, lexer.CloseParenToken)) {
			return p.parseCastExpressionGranular(parentNode);
		}

		/*
		
					case lexer.BacktickToken:
						return p.parseShellCommandExpression(parentNode);
		
					case lexer.OpenParenToken:
						// TODO
		//                return p.parseCastExpressionGranular(parentNode);
						break;*/

		// object-creation-expression (postfix-expression)
	case lexer.NewKeyword:
		return p.parseObjectCreationExpression(parentNode);

		// clone-expression (postfix-expression)
	case lexer.CloneKeyword:
		return p.parseCloneExpression(parentNode);

	case lexer.YieldKeyword,
		lexer.YieldFromKeyword:
		return p.parseYieldExpression(parentNode);

		// include-expression
		// include-once-expression
		// require-expression
		// require-once-expression
	case lexer.IncludeKeyword,
		lexer.IncludeOnceKeyword,
		lexer.RequireKeyword,
		lexer.RequireOnceKeyword:
		return p.parseScriptInclusionExpression(parentNode);
	}

	expression := p.parsePrimaryExpression(parentNode);
	return p.parsePostfixExpressionRest(expression);
}
}

func (p *Parser) parseBinaryExpressionOrHigher(precedence int, parentNode *ast.Node) ast.Node {
	leftOperand := p.parseUnaryExpressionOrHigher(parentNode);
}

func (p *Parser) getParseListElementFn(context ParseContext) func(*ast.Node) ast.Node {
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

func (p *Parser) parseStatementFn() func(*ast.Node) ast.Node {
	return func(parentNode *ast.Node) ast.Node {
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
				return ast.newSkippedToken(token)
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

func (p *Parser) parseStatement(parentNode ast.Node) *ast.Node {
	fn := p.parseStatementFn()
	st := fn(&parentNode)
	return &st
}

func (p *Parser) parseIfStatement(parentNode *ast.Node) ast.Node {
	st := ast.IfStatement{P: parentNode}
	st.IfKeyword = p.eat1(lexer.IfKeyword);
	st.OpenParen = p.eat1(lexer.OpenParenToken);
	exp := p.parseExpression(st, false);
	st.Expression = &exp;
	st.CloseParen = p.eat1(lexer.CloseParenToken);
	if (p.checkToken(lexer.ColonToken)) {
		st.Colon = p.eat1(lexer.ColonToken);
		st.Statements = p.parseList(st, IfClause2Elements);
	} else {
		// @todo
		st.Statements = []*ast.Node{p.parseStatement(st)};
	}
	st.ElseIfClauses = nil;
	for (p.checkToken(lexer.ElseIfKeyword)) {
		st.ElseIfClauses = append(st.ElseIfClauses, p.parseElseIfClause(st));
	}

	if (p.checkToken(lexer.ElseKeyword)) {
		st.ElseClause = p.parseElseClause(st);
	}

	st.EndifKeyword = p.eatOptional1(lexer.EndIfKeyword);
	if (st.EndifKeyword != nil) {
		st.SemiColon = p.eatSemicolonOrAbortStatement();
	}

	return st;
}

func (p *Parser) parseNamedLabelStatement(parentNode *ast.Node) ast.Node {
	st := ast.NamedLabelStatement{ P: parentNode}
	st.Name = p.eat1(lexer.Name)
	st.Colon = p.eat1(lexer.ColonToken)
	st.Statement = p.parseStatement(st)
	return st
}

func (p *Parser) parseCompoundStatement(parentNode *ast.Node) ast.Node {
	st := ast.CompoundStatement{ P: parentNode }
	st.OpenBrace = p.eat1(lexer.OpenBraceToken)
	st.Statements = p.parseList(st, BlockStatements)
	st.CloseBrace = p.eat1(lexer.CloseBraceToken)
	return st
}

func (p *Parser) lookahead(expectedKinds ...interface{}) bool {
	startPos := p.stream.Pos
	startToken := p.token
	succeeded := true
	for _, kind := range expectedKinds {
		token := p.stream.ScanNext()
		currentPos := p.stream.Pos
		eofPos := p.stream.EofPos

		switch kind.(type) {
		case []lexer.TokenKind:
			succeeded = false
			for _, kindOption := range kind.([]lexer.TokenKind) {
				if currentPos <= eofPos && token.Kind == kindOption {
					succeeded = true;
					break;
				}
			}
		case lexer.TokenKind:
			if currentPos > eofPos || token.Kind != kind {
				succeeded = false
				break
			}
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
	t := &lexer.Token{kind, token.FullStart, token.FullStart, 0, true}
	return t
}

func (p *Parser) parseExpression(parentNode ast.Node, force bool) ast.Node {
	token := p.token
	if token.Kind == lexer.EndOfFileToken {
		t := &lexer.Token{lexer.Expression, token.FullStart, token.FullStart, 0, true}
		missing := &ast.Missing{ &parentNode, t }
		return missing;
	}
	fnExpression := p.parseExpressionFn()
	expression := fnExpression(&parentNode)

	// @todo this not make sense
	// if (force && expression)

	return expression;
}
func (p *Parser) checkToken(kind lexer.TokenKind) bool {
	return p.token.Kind == kind
}

func (p *Parser) parseUnaryOpExpression(parent ast.Node) ast.Node {
	st := ast.UnaryOpExpression{}
	st.P = &parent
	st.Operator = p.eat(lexer.PlusToken, lexer.MinusToken, lexer.ExclamationToken, lexer.TildeToken)
	operand := p.parseUnaryExpressionOrHigher(st)
	st.Operand = &operand
	return st
}

func (p *Parser) eat(kinds ... lexer.TokenKind) *lexer.Token {
	token := p.token;
	for _, k := range kinds {
		if token.Kind == k {
			p.advanceToken()
			return &token
		}
	}
	t := &lexer.Token{kinds[0], token.FullStart, token.FullStart, 0, true}
	return t;
}

func (p *Parser) parseErrorControlExpression(parent ast.Node) ast.Node {
	errorExpr := ast.ErrorControlExpression{}
	errorExpr.P = &parent
	errorExpr.Operator = p.eat1(lexer.AtSymbolToken)
	operand := p.parseUnaryExpressionOrHigher(errorExpr)
	errorExpr.Operand = &operand
	return errorExpr
}

func (p *Parser) parsePrefixUpdateExpression(parent ast.Node) ast.Node {
	n := ast.PrefixUpdateExpression{}
	n.P = &parent
	n.IncrementOrDecrementOperator = p.eat(lexer.PlusPlusToken, lexer.MinusMinusToken)
	n.Operand = p.parsePrimaryExpression(n)
	switch n.Operand.(type) {
	case []ast.Variable:
		n.Operand = p.parsePostfixExpressionRest(n.Operand, false)
	}
	return n;
}

