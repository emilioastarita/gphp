package parser

import (
	"github.com/emilioastarita/gphp/ast"
	"github.com/emilioastarita/gphp/lexer"
)

type Parser struct {
	stream                            *lexer.TokensStream
	token                             *lexer.Token
	currentParseContext               ParseContext
	isParsingObjectCreationExpression bool
	reservedWordTokens                []lexer.TokenKind
	nameOrKeywordOrReservedWordTokens []lexer.TokenKind
	nameOrReservedWordTokens          []lexer.TokenKind
	parameterTypeDeclarationTokens    []lexer.TokenKind
	returnTypeDeclarationTokens       []lexer.TokenKind
	nameOrStaticOrReservedWordTokens  []lexer.TokenKind
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

func (p *Parser) ParseSourceFile(source string, uri string) *ast.SourceFileNode {

	typeDeclaration := []lexer.TokenKind{lexer.ArrayKeyword, lexer.CallableKeyword, lexer.BoolReservedWord,
		lexer.FloatReservedWord, lexer.IntReservedWord, lexer.StringReservedWord,
		lexer.ObjectReservedWord}

	p.returnTypeDeclarationTokens = []lexer.TokenKind{lexer.VoidReservedWord}
	p.returnTypeDeclarationTokens = append(p.returnTypeDeclarationTokens, typeDeclaration...)

	p.reservedWordTokens = lexer.GetReservedWords()
	p.nameOrStaticOrReservedWordTokens = []lexer.TokenKind{lexer.Name, lexer.StaticKeyword}
	p.nameOrStaticOrReservedWordTokens = append(p.nameOrStaticOrReservedWordTokens, p.reservedWordTokens...)

	p.parameterTypeDeclarationTokens = typeDeclaration
	p.nameOrKeywordOrReservedWordTokens = lexer.GetNameOrKeywordOrReservedWordTokens()
	p.nameOrReservedWordTokens = lexer.GetNameOrReservedWordTokens()
	p.stream = &lexer.TokensStream{}
	p.stream.Source(source)
	p.stream.CreateTokens()
	p.reset()
	sourceFile := &ast.SourceFileNode{P: nil, FileContents: source, Uri: uri}
	sourceFile.StatementList = make([]ast.Node, 0)
	if p.token.Kind != lexer.EndOfFileToken {
		sourceFile.Add(p.parseInlineHtml(sourceFile))
	}
	list := p.parseList(sourceFile, SourceElements)
	sourceFile.Merge(list)
	sourceFile.EndOfFileToken = p.eat1(lexer.EndOfFileToken)
	return sourceFile
}

func (p *Parser) reset() {
	p.advanceToken()
	p.currentParseContext = 0
}

func (p *Parser) advanceToken() {
	c := p.stream.ScanNext()
	p.token = c
}

func (p *Parser) parseInlineHtml(source ast.Node) ast.Node {
	end := p.eatOptional1(lexer.ScriptSectionEndTag)
	text := p.eatOptional1(lexer.InlineHtml)
	start := p.eatOptional1(lexer.ScriptSectionStartTag)
	n := &ast.InlineHtml{}
	n.P = source
	n.ScriptSectionEndTag = end
	n.ScriptSectionStartTag = start
	n.Text = text
	return n
}

func (p *Parser) eatOptional1(kind lexer.TokenKind) *lexer.Token {
	t := p.token
	if t.Kind == kind {
		p.advanceToken()
		return t
	}
	return nil
}

func (p *Parser) eatOptional(kinds ...lexer.TokenKind) *lexer.Token {
	t := p.token
	for _, kind := range kinds {
		if t.Kind == kind {
			p.advanceToken()
			return t
		}
	}
	return nil
}

func (p *Parser) parseList(parentNode ast.Node, listParseContext ParseContext) []ast.Node {
	savedParseContext := p.currentParseContext
	p.currentParseContext |= 1 << listParseContext
	parseListElementFn := p.getParseListElementFn(listParseContext)
	nodes := make([]ast.Node, 0)
	for p.isListTerminator(listParseContext) == false {
		if p.isValidListElement(listParseContext, p.token) {
			element := parseListElementFn(parentNode)
			element.SetParent(parentNode)
			nodes = append(nodes, element)
			continue
		}
		if p.isCurrentTokenValidInEnclosingContexts() {
			break
		}
		nodes = append(nodes, ast.NewSkippedNode(p.token))
		p.advanceToken()
	}
	p.currentParseContext = savedParseContext
	return nodes
}

func (p *Parser) parseClassElementFn() func(ast.Node) ast.Node {
	return func(parentNode ast.Node) ast.Node {
		modifiers := p.parseModifiers()
		token := p.token
		switch token.Kind {
		case lexer.ConstKeyword:
			return p.parseClassConstDeclaration(parentNode, modifiers)

		case lexer.FunctionKeyword:
			return p.parseMethodDeclaration(parentNode, modifiers)

		case lexer.VariableName:
			return p.parsePropertyDeclaration(parentNode, modifiers)

		case lexer.UseKeyword:
			return p.parseTraitUseClause(parentNode)

		default:
			missingClassMemberDeclaration := &ast.MissingMemberDeclaration{}
			missingClassMemberDeclaration.P = parentNode
			missingClassMemberDeclaration.Modifiers = modifiers
			return missingClassMemberDeclaration
		}
	}
}
func (p *Parser) parseExpressionFn() func(ast.Node) ast.Node {
	return func(parent ast.Node) ast.Node {
		return p.parseBinaryExpressionOrHigher(0, parent)
	}
}

func (p *Parser) parseModifiers() []*lexer.Token {
	modifiers := make([]*lexer.Token, 0)
	token := p.token
	for p.isModifier(token) {
		modifiers = append(modifiers, token)
		p.advanceToken()
		token = p.token
	}
	return modifiers
}

func (p *Parser) parseUnaryExpressionOrHigher(parentNode ast.Node) ast.Node {
	token := p.token
	switch token.Kind {
	case lexer.PlusToken,
		lexer.MinusToken,
		lexer.ExclamationToken,
		lexer.TildeToken:
		return p.parseUnaryOpExpression(parentNode)

		// error-control-expression
	case lexer.AtSymbolToken:
		return p.parseErrorControlExpression(parentNode)

		// prefix-increment-expression
	case lexer.PlusPlusToken,
		// prefix-decrement-expression
		lexer.MinusMinusToken:
		return p.parsePrefixUpdateExpression(parentNode)

	case lexer.ArrayCastToken,
		lexer.BoolCastToken,
		lexer.DoubleCastToken,
		lexer.IntCastToken,
		lexer.ObjectCastToken,
		lexer.StringCastToken,
		lexer.UnsetCastToken:
		return p.parseCastExpression(parentNode)

	case lexer.OpenParenToken:
		// TODO remove duplication
		if p.lookahead(
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
				lexer.UnsetKeyword}, lexer.CloseParenToken) {
			return p.parseCastExpressionGranular(parentNode)
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
		return p.parseObjectCreationExpression(parentNode)

		// clone-expression (postfix-expression)
	case lexer.CloneKeyword:
		return p.parseCloneExpression(parentNode)

	case lexer.YieldKeyword,
		lexer.YieldFromKeyword:
		return p.parseYieldExpression(parentNode)

		// include-expression
		// include-once-expression
		// require-expression
		// require-once-expression
	case lexer.IncludeKeyword,
		lexer.IncludeOnceKeyword,
		lexer.RequireKeyword,
		lexer.RequireOnceKeyword:
		return p.parseScriptInclusionExpression(parentNode)
	}

	expression := p.parsePrimaryExpression(parentNode)
	return p.parsePostfixExpressionRest(expression, true)
}

func (p *Parser) parseBinaryExpressionOrHigher(precedence int, parentNode ast.Node) ast.Node {
	leftOperand := p.parseUnaryExpressionOrHigher(parentNode)
	prevNewPrecedence, prevAssociativity := -1, ast.AssocUnknown
	for {
		token := p.token

		newPrecedence, associativity := p.getBinaryOperatorPrecedenceAndAssociativity(token)
		if prevAssociativity == ast.AssocNone && prevNewPrecedence == newPrecedence {
			break
		}
		shouldConsumeCurrentOperator := newPrecedence >= precedence
		if associativity != ast.AssocRight {
			shouldConsumeCurrentOperator = newPrecedence > precedence
		}

		if shouldConsumeCurrentOperator == false {
			break
		}
		unaryExpression, isUnaryExpression := leftOperand.(*ast.UnaryOpExpression)
		shouldOperatorTakePrecedenceOverUnary := token.Kind == lexer.AsteriskAsteriskToken && isUnaryExpression

		if shouldOperatorTakePrecedenceOverUnary {
			leftOperand = unaryExpression.Operand
		}
		p.advanceToken()

		var byRefToken *lexer.Token
		if token.Kind == lexer.EqualsToken {
			byRefToken = p.eatOptional1(lexer.AmpersandToken)
		}

		if token.Kind == lexer.QuestionToken {
			leftOperand = p.parseTernaryExpression(leftOperand, token)
		} else if token.Kind == lexer.EqualsToken {
			leftOperand = p.makeBinaryAssignmentExpression(leftOperand, token, byRefToken, p.parseBinaryExpressionOrHigher(newPrecedence, nil), parentNode)
		} else {
			leftOperand = p.makeBinaryExpression(leftOperand, token, p.parseBinaryExpressionOrHigher(newPrecedence, nil), parentNode)
		}

		if shouldOperatorTakePrecedenceOverUnary {
			leftOperand.SetParent(unaryExpression)
			unaryExpression.Operand = leftOperand
			leftOperand = unaryExpression
		}

		prevNewPrecedence = newPrecedence
		prevAssociativity = associativity
	}
	return leftOperand
}

func (p *Parser) parseSimpleVariableFn() func(ast.Node) ast.Node {
	return func(parentNode ast.Node) ast.Node {
		token := p.token
		variable := &ast.Variable{}
		variable.P = parentNode
		if token.Kind == lexer.DollarToken {
			variable.Dollar = p.eat1(lexer.DollarToken)
			token = p.token
			if token.Kind == lexer.OpenBraceToken {
				variable.Name = p.parseBracedExpression(variable)
			} else {
				variable.Name = p.parseSimpleVariable(variable)
			}

		} else if token.Kind == lexer.VariableName || token.Kind == lexer.StringVarname {
			// TODO consider splitting into dollar and name.
			// StringVarname is the variable name without , used in a template string e.g. `"{foo}"`
			tokName := p.eat(lexer.VariableName, lexer.StringVarname)
			tokNode := &ast.TokenNode{}
			tokNode.Token = tokName
			variable.Name = tokNode
		} else {
			variable.Name = ast.NewMissingToken(lexer.VariableName, token.FullStart, nil)
		}

		return variable
	}
}

func (p *Parser) parseBracedExpression(parentNode ast.Node) ast.Node {
	bracedExpression := &ast.BracedExpression{}
	bracedExpression.P = parentNode
	bracedExpression.OpenBrace = p.eat1(lexer.OpenBraceToken)
	bracedExpression.Expression = p.parseExpression(bracedExpression, false)
	bracedExpression.CloseBrace = p.eat1(lexer.CloseBraceToken)
	return bracedExpression
}

func (p *Parser) isExpressionStartFn() func(*lexer.Token) bool {

	return func(token *lexer.Token) bool {
		switch token.Kind {
		// Script Inclusion Expression
		case lexer.RequireKeyword,
			lexer.RequireOnceKeyword,
			lexer.IncludeKeyword,
			lexer.IncludeOnceKeyword,

			// yield-expression
			lexer.YieldKeyword,
			lexer.YieldFromKeyword,

			// object-creation-expression
			lexer.NewKeyword,
			lexer.CloneKeyword:
			return true

			// unary-op-expression
		case lexer.PlusToken,
			lexer.MinusToken,
			lexer.ExclamationToken,
			lexer.TildeToken,

			// error-control-expression
			lexer.AtSymbolToken,

			// prefix-increment-expression
			lexer.PlusPlusToken,
			// prefix-decrement-expression
			lexer.MinusMinusToken:
			return true

			// variable-name
		case lexer.VariableName,
			lexer.DollarToken:
			return true

			// qualified-name
		case lexer.Name,
			lexer.BackslashToken:
			return true
		case lexer.NamespaceKeyword:
			// TODO currently only supports qualified-names, but eventually parse namespace declarations
			return p.checkToken(lexer.BackslashToken)

			// literal
		case // TODO merge dec, oct, hex, bin, float . NumericLiteral
			lexer.OctalLiteralToken,
			lexer.HexadecimalLiteralToken,
			lexer.BinaryLiteralToken,
			lexer.FloatingLiteralToken,
			lexer.InvalidOctalLiteralToken,
			lexer.InvalidHexadecimalLiteral,
			lexer.InvalidBinaryLiteral,
			lexer.IntegerLiteralToken,

			lexer.StringLiteralToken,

			lexer.SingleQuoteToken,
			lexer.DoubleQuoteToken,
			lexer.HeredocStart,
			lexer.BacktickToken,

			// array-creation-expression
			lexer.ArrayKeyword,
			lexer.OpenBracketToken,

			// intrinsic-construct
			lexer.EchoKeyword,
			lexer.ListKeyword,
			lexer.UnsetKeyword,

			// intrinsic-operator
			lexer.EmptyKeyword,
			lexer.EvalKeyword,
			lexer.ExitKeyword,
			lexer.DieKeyword,
			lexer.IsSetKeyword,
			lexer.PrintKeyword,

			// ( expression )
			lexer.OpenParenToken,
			lexer.ArrayCastToken,
			lexer.BoolCastToken,
			lexer.DoubleCastToken,
			lexer.IntCastToken,
			lexer.ObjectCastToken,
			lexer.StringCastToken,
			lexer.UnsetCastToken,

			// anonymous-function-creation-expression
			lexer.StaticKeyword,
			lexer.FunctionKeyword:
			return true
		}
		return lexer.IsReservedWordToken(token.Kind)
	}
}

func (p *Parser) getParseListElementFn(context ParseContext) func(ast.Node) ast.Node {
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

func (p *Parser) parseStatementFn() func(ast.Node) ast.Node {
	return func(parentNode ast.Node) ast.Node {
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
		case lexer.ContinueKeyword, // continue-statement
			lexer.BreakKeyword: // break-statement
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
		case lexer.FinalKeyword,
			lexer.AbstractKeyword:
			if !p.lookahead(lexer.ClassKeyword) {
				p.advanceToken()
				return ast.NewSkippedNode(token)
			}
			return p.parseClassDeclaration(parentNode)
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

		expressionStatement := &ast.ExpressionStatement{}
		expressionStatement.P = parentNode

		ret := p.parseExpression(expressionStatement, true)

		_, isMissing := ret.(*ast.Missing)

		expressionStatement.Expression = []ast.Node{ret}
		if isMissing {
			p.advanceToken()
		}

		expressionStatement.Semicolon = p.eatSemicolonOrAbortStatement()
		return expressionStatement
	}
}

func (p *Parser) parseStatement(parentNode ast.Node) ast.Node {
	fn := p.parseStatementFn()
	st := fn(parentNode)
	return st
}

func (p *Parser) parseIfStatement(parentNode ast.Node) ast.Node {
	st := &ast.IfStatementNode{}
	st.P = parentNode
	st.IfKeyword = p.eat1(lexer.IfKeyword)
	st.OpenParen = p.eat1(lexer.OpenParenToken)
	exp := p.parseExpression(st, false)
	st.Expression = exp
	st.CloseParen = p.eat1(lexer.CloseParenToken)
	if p.checkToken(lexer.ColonToken) {
		st.Colon = p.eat1(lexer.ColonToken)
		st.Statements = p.parseList(st, IfClause2Elements)
	} else {
		// @todo
		st.Statements = p.parseStatement(st)
	}
	st.ElseIfClauses = make([]ast.Node, 0)
	for p.checkToken(lexer.ElseIfKeyword) {
		st.ElseIfClauses = append(st.ElseIfClauses, p.parseElseIfClause(st))
	}

	if p.checkToken(lexer.ElseKeyword) {
		st.ElseClause = p.parseElseClause(st)
	}

	st.EndifKeyword = p.eatOptional1(lexer.EndIfKeyword)
	if st.EndifKeyword != nil {
		st.Semicolon = p.eatSemicolonOrAbortStatement()
	}

	return st
}

func (p *Parser) parseNamedLabelStatement(parentNode ast.Node) ast.Node {
	st := &ast.NamedLabelStatement{}
	st.P = parentNode
	st.Name = p.eat1(lexer.Name)
	st.Colon = p.eat1(lexer.ColonToken)
	st.Statement = p.parseStatement(st)
	return st
}

func (p *Parser) parseCompoundStatement(parentNode ast.Node) ast.Node {
	st := &ast.CompoundStatementNode{}
	st.P = parentNode
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
					succeeded = true
					break
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
		return token
	}
	t := &lexer.Token{Kind: kind, FullStart: token.FullStart, Start: token.FullStart, Cat: lexer.TokenCatMissing}
	return t
}

func (p *Parser) parseExpression(parentNode ast.Node, force bool) ast.Node {
	token := p.token
	if token.Kind == lexer.EndOfFileToken {
		return ast.NewMissingToken(lexer.Expression, token.FullStart, parentNode)
	}
	fnExpression := p.parseExpressionFn()

	expression := fnExpression(parentNode)

	return expression
}

func (p *Parser) checkToken(kind lexer.TokenKind) bool {
	return p.token.Kind == kind
}

func (p *Parser) parseUnaryOpExpression(parent ast.Node) ast.Node {
	st := &ast.UnaryOpExpression{}
	st.P = parent
	st.Operator = p.eat(lexer.PlusToken, lexer.MinusToken, lexer.ExclamationToken, lexer.TildeToken)
	operand := p.parseUnaryExpressionOrHigher(st)
	st.Operand = operand
	return st
}

func (p *Parser) eat(kinds ...lexer.TokenKind) *lexer.Token {
	token := p.token
	for _, k := range kinds {
		if token.Kind == k {
			p.advanceToken()
			return token
		}
	}
	t := &lexer.Token{Kind: kinds[0], FullStart: token.FullStart, Start: token.FullStart, Cat: lexer.TokenCatMissing}
	return t
}

func (p *Parser) parseErrorControlExpression(parent ast.Node) ast.Node {
	errorExpr := &ast.ErrorControlExpression{}
	errorExpr.P = parent
	errorExpr.Operator = p.eat1(lexer.AtSymbolToken)
	operand := p.parseUnaryExpressionOrHigher(errorExpr)
	errorExpr.Operand = operand
	return errorExpr
}

func (p *Parser) parsePrefixUpdateExpression(parent ast.Node) ast.Node {
	n := &ast.PrefixUpdateExpression{}
	n.P = parent
	n.IncrementOrDecrementOperator = p.eat(lexer.PlusPlusToken, lexer.MinusMinusToken)
	op := p.parsePrimaryExpression(n)
	n.Operand = op
	switch n.Operand.(type) {
	case *ast.Missing:
		n.Operand = p.parsePostfixExpressionRest(n.Operand, false)
	}
	return n
}

func (p *Parser) parsePostfixExpressionRest(expression ast.Node, allowUpdateExpression bool) ast.Node {
	tokenKind := p.token.Kind

	// `--a++` is invalid
	if allowUpdateExpression &&
		(tokenKind == lexer.PlusPlusToken ||
			tokenKind == lexer.MinusMinusToken) {
		return p.parseParsePostfixUpdateExpression(expression)
	}

	retExpr := true
	switch expression.(type) {
	case *ast.Variable,
		*ast.ParenthesizedExpression,
		*ast.QualifiedName,
		*ast.CallExpression,
		*ast.MemberAccessExpression,
		*ast.SubscriptExpression,
		*ast.ScopedPropertyAccessExpression,
		*ast.StringLiteral,
		*ast.ArrayCreationExpression:
		retExpr = false
	}
	if retExpr {
		return expression
	}

	if tokenKind == lexer.ColonColonToken {
		expression = p.parseScopedPropertyAccessExpression(expression)
		return p.parsePostfixExpressionRest(expression, true)
	}

	for {
		tokenKind = p.token.Kind
		if tokenKind == lexer.OpenBraceToken ||
			tokenKind == lexer.OpenBracketToken {
			expression = p.parseSubscriptExpression(expression)
			return p.parsePostfixExpressionRest(expression, true)
		}

		switch expression.(type) {
		case *ast.ArrayCreationExpression:
			// Remaining postfix expressions are invalid, so abort
			return expression
		}

		if tokenKind == lexer.ArrowToken {
			expression = p.parseMemberAccessExpression(expression)
			return p.parsePostfixExpressionRest(expression, true)
		}

		if tokenKind == lexer.OpenParenToken && !p.isParsingObjectCreationExpression {
			expression = p.parseCallExpressionRest(expression)
			if p.checkToken(lexer.OpenParenToken) {
				// a()() should get parsed as CallExpr-ParenExpr, so do not recurse
				return expression
			}
			return p.parsePostfixExpressionRest(expression, true)
		}

		// Reached the end of the postfix-expression, so return
		return expression
	}
}

func (p *Parser) parseParsePostfixUpdateExpression(prefixExpression ast.Node) ast.Node {
	postfixUpdateExpression := &ast.PostfixUpdateExpression{}
	postfixUpdateExpression.Operand = prefixExpression
	postfixUpdateExpression.P = prefixExpression.Parent()
	prefixExpression.SetParent(postfixUpdateExpression)
	postfixUpdateExpression.IncrementOrDecrementOperator =
		p.eat(lexer.PlusPlusToken, lexer.MinusMinusToken)
	return postfixUpdateExpression
}

func (p *Parser) parsePrimaryExpression(parentNode ast.Node) ast.Node {
	token := p.token
	switch token.Kind {
	// variable-name
	case lexer.VariableName, // TODO special case this
		lexer.DollarToken:
		return p.parseSimpleVariable(parentNode)

		// qualified-name
	case lexer.Name, // TODO Qualified name
		lexer.BackslashToken,
		lexer.NamespaceKeyword:
		return p.parseQualifiedName(parentNode)

	case // TODO merge dec, oct, hex, bin, float . NumericLiteral
		lexer.OctalLiteralToken,
		lexer.HexadecimalLiteralToken,
		lexer.BinaryLiteralToken,
		lexer.FloatingLiteralToken,
		lexer.InvalidOctalLiteralToken,
		lexer.InvalidHexadecimalLiteral,
		lexer.InvalidBinaryLiteral,
		lexer.IntegerLiteralToken:
		return p.parseNumericLiteralExpression(parentNode)

	case lexer.StringLiteralToken:
		return p.parseStringLiteralExpression(parentNode)

	case lexer.DoubleQuoteToken,
		lexer.SingleQuoteToken,
		lexer.HeredocStart,
		lexer.BacktickToken:
		return p.parseStringLiteralExpression2(parentNode)

		// TODO constant-expression

		// array-creation-expression
	case lexer.ArrayKeyword,
		lexer.OpenBracketToken:
		return p.parseArrayCreationExpression(parentNode)

		// intrinsic-construct
	case lexer.EchoKeyword:
		return p.parseEchoExpression(parentNode)
	case lexer.ListKeyword:
		return p.parseListIntrinsicExpression(parentNode)
	case lexer.UnsetKeyword:
		return p.parseUnsetIntrinsicExpression(parentNode)

		// intrinsic-operator
	case lexer.EmptyKeyword:
		return p.parseEmptyIntrinsicExpression(parentNode)
	case lexer.EvalKeyword:
		return p.parseEvalIntrinsicExpression(parentNode)

	case lexer.ExitKeyword,
		lexer.DieKeyword:
		return p.parseExitIntrinsicExpression(parentNode)

	case lexer.IsSetKeyword:
		return p.parseIssetIntrinsicExpression(parentNode)

	case lexer.PrintKeyword:
		return p.parsePrintIntrinsicExpression(parentNode)

		// ( expression )
	case lexer.OpenParenToken:
		return p.parseParenthesizedExpression(parentNode)

		// anonymous-function-creation-expression
	case lexer.StaticKeyword:
		// handle `static::`, `static(`
		if p.lookahead([]lexer.TokenKind{lexer.ColonColonToken, lexer.OpenParenToken}) || (!p.lookahead(lexer.FunctionKeyword)) {
			return p.parseQualifiedName(parentNode)
		}
		// Could be `static function` anonymous function creation expression, so flow through
		return p.parseAnonymousFunctionCreationExpression(parentNode)
	case lexer.FunctionKeyword:
		return p.parseAnonymousFunctionCreationExpression(parentNode)

	case lexer.TrueReservedWord,
		lexer.FalseReservedWord,
		lexer.NullReservedWord:
		// handle `true::`, `true(`, `true\`
		if p.lookahead([]lexer.TokenKind{lexer.BackslashToken, lexer.ColonColonToken, lexer.OpenParenToken}) {
			return p.parseQualifiedName(parentNode)
		}
		return p.parseReservedWordExpression(parentNode)
	}

	if lexer.IsReservedWordToken(token.Kind) {
		return p.parseQualifiedName(parentNode)
	}
	return ast.NewMissingToken(lexer.Expression, token.FullStart, parentNode)
}

func (p *Parser) parseSimpleVariable(variable ast.Node) ast.Node {
	fn := p.parseSimpleVariableFn()
	return fn(variable)
}

func (p *Parser) isModifier(token *lexer.Token) bool {
	switch token.Kind {
	// class-modifier
	case lexer.AbstractKeyword,
		lexer.FinalKeyword,
		// visibility-modifier
		lexer.PublicKeyword,
		lexer.ProtectedKeyword,
		lexer.PrivateKeyword,
		// static-modifier
		lexer.StaticKeyword,
		// var
		lexer.VarKeyword:
		return true
	}
	return false
}

func (p *Parser) parseClassConstDeclaration(parentNode ast.Node, modifiers []*lexer.Token) ast.Node {
	classConstDeclaration := &ast.ClassConstDeclaration{}
	classConstDeclaration.P = parentNode
	classConstDeclaration.Modifiers = modifiers
	classConstDeclaration.ConstKeyword = p.eat1(lexer.ConstKeyword)
	classConstDeclaration.ConstElements = p.parseConstElements(classConstDeclaration)
	classConstDeclaration.Semicolon = p.eat1(lexer.SemicolonToken)
	return classConstDeclaration
}

func (p *Parser) parseConstElements(parentNode ast.Node) ast.Node {
	constList := &ast.ConstElementList{}
	fn := func(token *lexer.Token) bool {
		return lexer.IsNameOrKeywordOrReservedWordTokens(token.Kind)
	}
	return p.parseDelimitedList(
		constList,
		lexer.CommaToken,
		fn,
		p.parseConstElementFn(),
		parentNode,
		false,
	)
}

func (p *Parser) parseMethodDeclaration(parentNode ast.Node, modifiers []*lexer.Token) ast.Node {
	methodDeclaration := &ast.MethodDeclaration{}
	methodDeclaration.Modifiers = modifiers
	p.parseFunctionType(methodDeclaration, true, false)
	methodDeclaration.P = parentNode
	return methodDeclaration
}

func (p *Parser) parseFunctionType(functionDeclaration ast.FunctionInterface, canBeAbstract bool, isAnonymous bool) {
	functionDeclaration.SetFunctionKeyword(p.eat1(lexer.FunctionKeyword))
	functionDeclaration.SetByRefToken(p.eatOptional1(lexer.AmpersandToken))

	if isAnonymous {
		t := &ast.TokenNode{Token: p.eatOptional(p.nameOrKeywordOrReservedWordTokens...)}
		functionDeclaration.SetName(t)
	} else {
		t := &ast.TokenNode{Token: p.eat(p.nameOrKeywordOrReservedWordTokens...)}
		functionDeclaration.SetName(t)
	}

	hasNameToken := functionDeclaration.GetName() != nil && functionDeclaration.GetName().GetToken() != nil

	if hasNameToken {
		functionDeclaration.GetName().GetToken().Kind = lexer.Name
	}

	if isAnonymous && hasNameToken {
		// Anonymous functions should not have names
		functionDeclaration.SetName(ast.NewSkippedNode(functionDeclaration.GetName().GetToken())) // TODO instaed handle this during post-walk
	}

	functionDeclaration.SetOpenParen(p.eat1(lexer.OpenParenToken))
	parameterDeclaration := &ast.ParameterDeclarationList{}
	functionDeclaration.SetParameters(p.parseDelimitedList(
		parameterDeclaration,
		lexer.CommaToken,
		p.isParameterStartFn(),
		p.parseParameterFn(),
		functionDeclaration, false))

	functionDeclaration.SetCloseParen(p.eat1(lexer.CloseParenToken))

	if isAnonymous {
		switch val := functionDeclaration.(type) {
		case *ast.AnonymousFunctionCreationExpression:
			val.AnonymousFunctionUseClause = p.parseAnonymousFunctionUseClause(val)
		}
	}

	if p.checkToken(lexer.ColonToken) {
		functionDeclaration.SetColonToken(p.eat1(lexer.ColonToken))
		functionDeclaration.SetQuestionToken(p.eatOptional1(lexer.QuestionToken))
		functionDeclaration.SetReturnType(p.parseReturnTypeDeclaration(functionDeclaration))
	}

	var tokNode *ast.TokenNode
	if canBeAbstract {
		tokNode = &ast.TokenNode{Token: p.eatOptional1(lexer.SemicolonToken)}
		functionDeclaration.SetCompoundStatementOrSemicolon(tokNode)
	}

	if tokNode == nil || tokNode.Token == nil {
		functionDeclaration.SetCompoundStatementOrSemicolon(p.parseCompoundStatement(functionDeclaration))
	}
}

func (p *Parser) parsePropertyDeclaration(parentNode ast.Node, modifiers []*lexer.Token) ast.Node {
	propertyDeclaration := &ast.PropertyDeclaration{}
	propertyDeclaration.P = parentNode
	propertyDeclaration.Modifiers = modifiers
	propertyDeclaration.PropertyElements = p.parseExpressionList(propertyDeclaration)
	propertyDeclaration.Semicolon = p.eat1(lexer.SemicolonToken)
	return propertyDeclaration
}

func (p *Parser) parseExpressionList(parentNode ast.Node) ast.DelimitedList {
	expressionList := &ast.ExpressionList{}
	return p.parseDelimitedList(expressionList, lexer.CommaToken, p.isExpressionStartFn(), p.parseExpressionFn(), parentNode, false)
}

type ElementStartFn func(*lexer.Token) bool

type ParseElementFn func(ast.Node) ast.Node

func (p *Parser) parseDelimitedList(node ast.DelimitedList, delimiter lexer.TokenKind, isElementStartFn ElementStartFn, parseElementFn ParseElementFn, parentNode ast.Node, allowEmptyElements bool) ast.DelimitedList {
	// TODO consider allowing empty delimiter to be more tolerant
	token := p.token
	for {
		if isElementStartFn(token) {
			r := parseElementFn(node)
			node.AddNode(r)
		} else if !allowEmptyElements || (allowEmptyElements && !p.checkToken(delimiter)) {
			break
		}
		delimeterToken := p.eatOptional(delimiter)
		if delimeterToken != nil {
			tokNod := &ast.TokenNode{Token: delimeterToken}
			node.AddNode(tokNod)
		}
		token = p.token
		// TODO ERROR CASE - no delimeter, but a param follows
		if delimeterToken == nil {
			break
		}
	}

	node.SetParent(parentNode)
	if node.Children() == nil {
		return nil
	}
	return node
}

func (p *Parser) parseTraitUseClause(parentNode ast.Node) ast.Node {
	traitUseClause := &ast.TraitUseClause{}
	traitUseClause.P = parentNode
	traitUseClause.UseKeyword = p.eat1(lexer.UseKeyword)
	traitUseClause.TraitNameList = p.parseQualifiedNameList(traitUseClause)
	traitUseClause.SemicolonOrOpenBrace = p.eat(lexer.OpenBraceToken, lexer.SemicolonToken)
	if traitUseClause.SemicolonOrOpenBrace.Kind == lexer.OpenBraceToken {
		traitUseClause.TraitSelectAndAliasClauses = p.parseTraitSelectAndAliasClauseList(traitUseClause)
		traitUseClause.CloseBrace = p.eat1(lexer.CloseBraceToken)
	}

	return traitUseClause
}

func (p *Parser) parseCastExpression(parentNode ast.Node) ast.Node {
	castExpression := &ast.CastExpression{}
	castExpression.P = parentNode
	castExpression.CastType = p.eat(
		lexer.ArrayCastToken,
		lexer.BoolCastToken,
		lexer.DoubleCastToken,
		lexer.IntCastToken,
		lexer.ObjectCastToken,
		lexer.StringCastToken,
		lexer.UnsetCastToken)
	castExpression.Operand = p.parseUnaryExpressionOrHigher(castExpression)
	return castExpression
}

func (p *Parser) parseCastExpressionGranular(parentNode ast.Node) ast.Node {
	castExpression := &ast.CastExpression{}
	castExpression.P = parentNode
	castExpression.OpenParen = p.eat1(lexer.OpenParenToken)
	castExpression.CastType = p.eat(
		lexer.ArrayKeyword,
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
		lexer.UnsetKeyword)
	castExpression.CloseParen = p.eat1(lexer.CloseParenToken)
	castExpression.Operand = p.parseUnaryExpressionOrHigher(castExpression)
	return castExpression
}

func (p *Parser) parseObjectCreationExpression(parentNode ast.Node) ast.Node {
	objectCreationExpression := &ast.ObjectCreationExpression{}
	objectCreationExpression.P = parentNode
	objectCreationExpression.NewKeword = p.eat1(lexer.NewKeyword)

	// TODO - add tests for this scenario
	p.isParsingObjectCreationExpression = true
	if r := p.eatOptional1(lexer.ClassKeyword); r != nil {
		tokNode := &ast.TokenNode{}
		tokNode.Token = r
		objectCreationExpression.ClassTypeDesignator = tokNode
	} else {
		r := p.parseExpression(objectCreationExpression, false)
		objectCreationExpression.ClassTypeDesignator = r
	}
	p.isParsingObjectCreationExpression = false

	objectCreationExpression.OpenParen = p.eatOptional1(lexer.OpenParenToken)

	if objectCreationExpression.OpenParen != nil {
		objectCreationExpression.ArgumentExpressionList = p.parseArgumentExpressionList(objectCreationExpression)
		objectCreationExpression.CloseParen = p.eat1(lexer.CloseParenToken)
	}

	objectCreationExpression.ClassBaseClause = p.parseClassBaseClause(objectCreationExpression)
	objectCreationExpression.ClassInterfaceClause = p.parseClassInterfaceClause(objectCreationExpression)

	if p.token.Kind == lexer.OpenBraceToken {
		objectCreationExpression.ClassMembers = p.parseClassMembers(objectCreationExpression)
	}

	return objectCreationExpression
}

func (p *Parser) parseArgumentExpressionList(parentNode ast.Node) ast.Node {
	argumentExpressionList := &ast.ArgumentExpressionList{}
	return p.parseDelimitedList(
		argumentExpressionList,
		lexer.CommaToken,
		p.isArgumentExpressionStartFn(),
		p.parseArgumentExpressionFn(),
		parentNode,
		false,
	)
}

func (p *Parser) parseClassBaseClause(parentNode ast.Node) ast.Node {
	classBaseClause := &ast.ClassBaseClause{}
	classBaseClause.P = parentNode
	classBaseClause.ExtendsKeyword = p.eatOptional1(lexer.ExtendsKeyword)
	if classBaseClause.ExtendsKeyword == nil {
		return nil
	}
	classBaseClause.BaseClass = p.parseQualifiedName(classBaseClause)
	return classBaseClause
}

func (p *Parser) parseQualifiedName(parentNode ast.Node) ast.Node {
	return (p.parseQualifiedNameFn())(parentNode)
}

func (p *Parser) parseClassInterfaceClause(parentNode ast.Node) ast.Node {
	classInterfaceClause := &ast.ClassInterfaceClause{}
	classInterfaceClause.P = parentNode
	classInterfaceClause.ImplementsKeyword = p.eatOptional1(lexer.ImplementsKeyword)
	if classInterfaceClause.ImplementsKeyword == nil {
		return nil
	}

	classInterfaceClause.InterfaceNameList = p.parseQualifiedNameList(classInterfaceClause)
	return classInterfaceClause
}

func (p *Parser) parseQualifiedNameList(parentNode ast.Node) ast.Node {
	qualifiedNameList := &ast.QualifiedNameList{}
	return p.parseDelimitedList(
		qualifiedNameList,
		lexer.CommaToken,
		p.isQualifiedNameStartFn(),
		p.parseQualifiedNameFn(),
		parentNode, false)
}

func (p *Parser) parseCloneExpression(parentNode ast.Node) ast.Node {
	cloneExpression := &ast.CloneExpression{}
	cloneExpression.P = parentNode
	cloneExpression.CloneKeyword = p.eat1(lexer.CloneKeyword)
	cloneExpression.Expression = p.parseUnaryExpressionOrHigher(cloneExpression)
	return cloneExpression
}

func (p *Parser) parseYieldExpression(parentNode ast.Node) ast.Node {
	yieldExpression := &ast.YieldExpression{}
	yieldExpression.P = parentNode
	yieldExpression.YieldOrYieldFromKeyword = p.eat(
		lexer.YieldFromKeyword,
		lexer.YieldKeyword,
	)

	yieldExpression.ArrayElement = p.parseArrayElement(yieldExpression)
	return yieldExpression
}

func (p *Parser) parseScriptInclusionExpression(parentNode ast.Node) ast.Node {
	scriptInclusionExpression := &ast.ScriptInclusionExpression{}
	scriptInclusionExpression.P = parentNode
	scriptInclusionExpression.RequireOrIncludeKeyword = p.eat(
		lexer.RequireKeyword, lexer.RequireOnceKeyword,
		lexer.IncludeKeyword, lexer.IncludeOnceKeyword)
	scriptInclusionExpression.Expression = p.parseExpression(scriptInclusionExpression, false)
	return scriptInclusionExpression
}

func (p *Parser) parseTraitElementFn() func(ast.Node) ast.Node {
	return func(parentNode ast.Node) ast.Node {
		modifiers := p.parseModifiers()
		token := p.token
		switch token.Kind {
		case lexer.FunctionKeyword:
			return p.parseMethodDeclaration(parentNode, modifiers)

		case lexer.VariableName:
			return p.parsePropertyDeclaration(parentNode, modifiers)

		case lexer.UseKeyword:
			return p.parseTraitUseClause(parentNode)

		default:
			missingTraitMemberDeclaration := &ast.MissingMemberDeclaration{}
			missingTraitMemberDeclaration.P = parentNode
			missingTraitMemberDeclaration.Modifiers = modifiers
			return missingTraitMemberDeclaration
		}
	}
}

func (p *Parser) parseInterfaceElementFn() func(ast.Node) ast.Node {
	return func(parentNode ast.Node) ast.Node {
		modifiers := p.parseModifiers()
		token := p.token
		switch token.Kind {
		case lexer.ConstKeyword:
			return p.parseClassConstDeclaration(parentNode, modifiers)

		case lexer.FunctionKeyword:
			return p.parseMethodDeclaration(parentNode, modifiers)

		default:
			missingInterfaceMemberDeclaration := &ast.MissingMemberDeclaration{}
			missingInterfaceMemberDeclaration.P = parentNode
			missingInterfaceMemberDeclaration.Modifiers = modifiers
			return missingInterfaceMemberDeclaration
		}
	}
}

func (p *Parser) parseCaseOrDefaultStatement() func(ast.Node) ast.Node {
	return func(parentNode ast.Node) ast.Node {
		caseStatement := &ast.CaseStatementNode{}
		caseStatement.P = parentNode
		// TODO add error checking
		caseStatement.CaseKeyword = p.eat(lexer.CaseKeyword, lexer.DefaultKeyword)
		if caseStatement.CaseKeyword.Kind == lexer.CaseKeyword {
			expr := p.parseExpression(caseStatement, false)
			caseStatement.Expression = expr
		}
		caseStatement.DefaultLabelTerminator = p.eat(lexer.ColonToken, lexer.SemicolonToken)
		caseStatement.StatementList = p.parseList(caseStatement, CaseStatementElements)
		return caseStatement
	}
}

func (p *Parser) parseSwitchStatement(parentNode ast.Node) ast.Node {
	switchStatement := &ast.SwitchStatementNode{}
	switchStatement.P = parentNode
	switchStatement.SwitchKeyword = p.eat1(lexer.SwitchKeyword)
	switchStatement.OpenParen = p.eat1(lexer.OpenParenToken)
	expr := p.parseExpression(switchStatement, false)
	switchStatement.Expression = expr
	switchStatement.CloseParen = p.eat1(lexer.CloseParenToken)
	switchStatement.OpenBrace = p.eatOptional1(lexer.OpenBraceToken)
	switchStatement.Colon = p.eatOptional1(lexer.ColonToken)
	switchStatement.CaseStatements = p.parseList(switchStatement, SwitchStatementElements)
	if switchStatement.Colon != nil {
		switchStatement.Endswitch = p.eat1(lexer.EndSwitchKeyword)
		switchStatement.Semicolon = p.eatSemicolonOrAbortStatement()
	} else {
		switchStatement.CloseBrace = p.eat1(lexer.CloseBraceToken)
	}

	return switchStatement
}

func (p *Parser) eatSemicolonOrAbortStatement() *lexer.Token {
	if p.token.Kind != lexer.ScriptSectionEndTag {
		return p.eat1(lexer.SemicolonToken)
	}
	return nil
}

func (p *Parser) parseWhileStatement(parentNode ast.Node) ast.Node {
	whileStatement := &ast.WhileStatement{}
	whileStatement.P = parentNode
	whileStatement.WhileToken = p.eat1(lexer.WhileKeyword)
	whileStatement.OpenParen = p.eat1(lexer.OpenParenToken)
	expr := p.parseExpression(whileStatement, false)
	whileStatement.Expression = expr
	whileStatement.CloseParen = p.eat1(lexer.CloseParenToken)
	whileStatement.Colon = p.eatOptional1(lexer.ColonToken)
	if whileStatement.Colon != nil {
		whileStatement.Statements = p.parseList(whileStatement, WhileStatementElements)
		whileStatement.EndWhile = p.eat1(lexer.EndWhileKeyword)
		whileStatement.Semicolon = p.eatSemicolonOrAbortStatement()
	} else {
		whileStatement.Statements = p.parseStatement(whileStatement)
	}
	return whileStatement
}

func (p *Parser) parseDoStatement(parentNode ast.Node) ast.Node {
	doStatement := &ast.DoStatement{}
	doStatement.P = parentNode
	doStatement.Do = p.eat1(lexer.DoKeyword)
	doStatement.Statement = p.parseStatement(doStatement)
	doStatement.WhileToken = p.eat1(lexer.WhileKeyword)
	doStatement.OpenParen = p.eat1(lexer.OpenParenToken)
	expr := p.parseExpression(doStatement, false)
	doStatement.Expression = expr
	doStatement.CloseParen = p.eat1(lexer.CloseParenToken)
	doStatement.Semicolon = p.eatSemicolonOrAbortStatement()
	return doStatement
}

func (p *Parser) parseForStatement(parentNode ast.Node) ast.Node {
	forStatement := &ast.ForStatement{}
	forStatement.P = parentNode
	forStatement.For = p.eat1(lexer.ForKeyword)
	forStatement.OpenParen = p.eat1(lexer.OpenParenToken)
	forStatement.ForInitializer = p.parseExpressionList(forStatement) // TODO spec is redundant
	forStatement.ExprGroupSemicolon1 = p.eat1(lexer.SemicolonToken)
	forStatement.ForControl = p.parseExpressionList(forStatement)
	forStatement.ExprGroupSemicolon2 = p.eat1(lexer.SemicolonToken)
	forStatement.ForEndOfLoop = p.parseExpressionList(forStatement)
	forStatement.CloseParen = p.eat1(lexer.CloseParenToken)
	forStatement.Colon = p.eatOptional1(lexer.ColonToken)
	if forStatement.Colon != nil {
		forStatement.Statements = p.parseList(forStatement, ForStatementElements)
		forStatement.EndFor = p.eat1(lexer.EndForKeyword)
		forStatement.EndForSemicolon = p.eatSemicolonOrAbortStatement()
	} else {
		forStatement.Statements = p.parseStatement(forStatement)
	}
	return forStatement
}

func (p *Parser) parseForeachStatement(parentNode ast.Node) ast.Node {
	foreachStatement := &ast.ForeachStatement{}
	foreachStatement.P = parentNode
	foreachStatement.Foreach = p.eat1(lexer.ForeachKeyword)
	foreachStatement.OpenParen = p.eat1(lexer.OpenParenToken)
	expr := p.parseExpression(foreachStatement, false)
	foreachStatement.ForEachCollectionName = expr
	foreachStatement.AsKeyword = p.eat1(lexer.AsKeyword)
	foreachStatement.ForeachKey = p.tryParseForeachKey(foreachStatement)
	foreachStatement.ForeachValue = p.parseForeachValue(foreachStatement)
	foreachStatement.CloseParen = p.eat1(lexer.CloseParenToken)
	foreachStatement.Colon = p.eatOptional1(lexer.ColonToken)
	if foreachStatement.Colon != nil {
		foreachStatement.Statements = p.parseList(foreachStatement, ForeachStatementElements)
		foreachStatement.EndForeach = p.eat1(lexer.EndForEachKeyword)
		foreachStatement.EndForeachSemicolon = p.eatSemicolonOrAbortStatement()
	} else {
		foreachStatement.Statements = p.parseStatement(foreachStatement)
	}
	return foreachStatement
}

func (p *Parser) tryParseForeachKey(parentNode ast.Node) ast.Node {
	if !p.isExpressionStart(p.token) {
		return nil
	}

	startPos := p.stream.Pos
	startToken := p.token
	foreachKey := &ast.ForeachKey{}
	foreachKey.P = parentNode
	foreachKey.Expression = p.parseExpression(foreachKey, false)
	if !p.checkToken(lexer.DoubleArrowToken) {
		p.stream.Pos = startPos
		p.token = startToken
		return nil
	}

	foreachKey.Arrow = p.eat1(lexer.DoubleArrowToken)
	return foreachKey
}

func (p *Parser) parseForeachValue(parentNode ast.Node) ast.Node {
	foreachValue := &ast.ForeachValue{}
	foreachValue.P = parentNode
	foreachValue.Ampersand = p.eatOptional1(lexer.AmpersandToken)
	foreachValue.Expression = p.parseExpression(foreachValue, false)
	return foreachValue
}

func (p *Parser) isExpressionStart(token *lexer.Token) bool {
	fn := p.isExpressionStartFn()
	return fn(token)
}

func (p *Parser) parseEmptyIntrinsicExpression(parentNode ast.Node) ast.Node {
	emptyExpression := &ast.EmptyIntrinsicExpression{}
	emptyExpression.P = parentNode
	emptyExpression.EmptyKeyword = p.eat1(lexer.EmptyKeyword)
	emptyExpression.OpenParen = p.eat1(lexer.OpenParenToken)
	emptyExpression.Expression = p.parseExpression(emptyExpression, false)
	emptyExpression.CloseParen = p.eat1(lexer.CloseParenToken)
	return emptyExpression
}

func (p *Parser) parseGotoStatement(parentNode ast.Node) ast.Node {
	gotoStatement := &ast.GotoStatement{}
	gotoStatement.P = parentNode
	gotoStatement.Goto = p.eat1(lexer.GotoKeyword)
	gotoStatement.Name = p.eat1(lexer.Name)
	gotoStatement.Semicolon = p.eatSemicolonOrAbortStatement()
	return gotoStatement
}

func (p *Parser) parseBreakOrContinueStatement(parentNode ast.Node) ast.Node {
	// TODO should be error checking if on top level
	continueStatement := &ast.BreakOrContinueStatement{}
	continueStatement.P = parentNode
	continueStatement.BreakOrContinueKeyword = p.eat(lexer.ContinueKeyword, lexer.BreakKeyword)

	if p.isExpressionStart(p.token) {
		continueStatement.BreakoutLevel = p.parseExpression(continueStatement, false)
	}

	continueStatement.Semicolon = p.eatSemicolonOrAbortStatement()

	return continueStatement

}

func (p *Parser) parseReturnStatement(parentNode ast.Node) ast.Node {
	returnStatement := &ast.ReturnStatement{}
	returnStatement.P = parentNode
	returnStatement.ReturnKeyword = p.eat1(lexer.ReturnKeyword)
	if p.isExpressionStart(p.token) {
		returnStatement.Expression = p.parseExpression(returnStatement, false)
	}
	returnStatement.Semicolon = p.eatSemicolonOrAbortStatement()
	return returnStatement
}

func (p *Parser) parseThrowStatement(parentNode ast.Node) ast.Node {
	throwStatement := &ast.ThrowStatement{}
	throwStatement.P = parentNode
	throwStatement.ThrowKeyword = p.eat1(lexer.ThrowKeyword)
	// TODO error for failures to parse expressions when not optional
	throwStatement.Expression = p.parseExpression(throwStatement, false)
	throwStatement.Semicolon = p.eatSemicolonOrAbortStatement()

	return throwStatement
}

func (p *Parser) parseTryStatement(parentNode ast.Node) ast.Node {
	tryStatement := &ast.TryStatement{}
	tryStatement.P = parentNode
	tryStatement.TryKeyword = p.eat1(lexer.TryKeyword)
	tryStatement.CompoundStatement = p.parseCompoundStatement(tryStatement) // TODO verifiy this is only compound

	tryStatement.CatchClauses = make([]ast.Node, 0) // TODO - should be some standard for empty arrays vs. null?
	for p.checkToken(lexer.CatchKeyword) {
		tryStatement.CatchClauses = append(tryStatement.CatchClauses, p.parseCatchClause(tryStatement))
	}

	if p.checkToken(lexer.FinallyKeyword) {
		tryStatement.FinallyClause = p.parseFinallyClause(tryStatement)
	}

	return tryStatement
}

func (p *Parser) parseDeclareStatement(parentNode ast.Node) ast.Node {
	declareStatement := &ast.DeclareStatement{}
	declareStatement.P = parentNode
	declareStatement.DeclareKeyword = p.eat1(lexer.DeclareKeyword)
	declareStatement.OpenParen = p.eat1(lexer.OpenParenToken)
	declareStatement.DeclareDirective = p.parseDeclareDirective(declareStatement)
	declareStatement.CloseParen = p.eat1(lexer.CloseParenToken)

	if p.checkToken(lexer.SemicolonToken) {
		declareStatement.Semicolon = p.eatSemicolonOrAbortStatement()
	} else if p.checkToken(lexer.ColonToken) {
		declareStatement.Colon = p.eat1(lexer.ColonToken)
		declareStatement.Statements = p.parseList(declareStatement, DeclareStatementElements)
		declareStatement.EnddeclareKeyword = p.eat1(lexer.EndDeclareKeyword)
		declareStatement.Semicolon = p.eatSemicolonOrAbortStatement()
	} else {
		declareStatement.Statements = p.parseStatement(declareStatement)
	}

	return declareStatement

}

func (p *Parser) parseFunctionDeclaration(parentNode ast.Node) ast.Node {
	functionNode := &ast.FunctionDeclaration{}
	p.parseFunctionType(functionNode, false, false)
	functionNode.P = parentNode
	return functionNode
}

func (p *Parser) parseClassMembers(parentNode ast.Node) ast.Node {
	classMembers := &ast.ClassMembersNode{}
	classMembers.OpenBrace = p.eat1(lexer.OpenBraceToken)
	classMembers.ClassMemberDeclarations = p.parseList(classMembers, ClassMembers)
	classMembers.CloseBrace = p.eat1(lexer.CloseBraceToken)
	classMembers.P = parentNode
	return classMembers
}

func (p *Parser) parseClassDeclaration(parentNode ast.Node) ast.Node {
	classNode := &ast.ClassDeclaration{} // TODO verify not nested
	classNode.P = parentNode
	classNode.AbstractOrFinalModifier = p.eatOptional(lexer.AbstractKeyword, lexer.FinalKeyword)
	classNode.ClassKeyword = p.eat1(lexer.ClassKeyword)
	classNode.Name = p.eat(p.nameOrReservedWordTokens...) // TODO should be any
	classNode.Name.Kind = lexer.Name
	classNode.ClassBaseClause = p.parseClassBaseClause(classNode)
	classNode.ClassInterfaceClause = p.parseClassInterfaceClause(classNode)
	classNode.ClassMembers = p.parseClassMembers(classNode)
	return classNode

}

func (p *Parser) parseInterfaceDeclaration(parentNode ast.Node) ast.Node {
	interfaceDeclaration := &ast.InterfaceDeclaration{} // TODO verify not nested
	interfaceDeclaration.P = parentNode
	interfaceDeclaration.InterfaceKeyword = p.eat1(lexer.InterfaceKeyword)
	interfaceDeclaration.Name = p.eat1(lexer.Name)
	interfaceDeclaration.InterfaceBaseClause = p.parseInterfaceBaseClause(interfaceDeclaration)
	interfaceDeclaration.InterfaceMembers = p.parseInterfaceMembers(interfaceDeclaration)
	return interfaceDeclaration

}

func (p *Parser) parseNamespaceDefinition(parentNode ast.Node) ast.Node {
	namespaceDefinition := &ast.NamespaceDefinition{}
	namespaceDefinition.P = parentNode

	namespaceDefinition.NamespaceKeyword = p.eat1(lexer.NamespaceKeyword)
	if !p.checkToken(lexer.NamespaceKeyword) {
		namespaceDefinition.Name = p.parseQualifiedName(namespaceDefinition) // TODO only optional with compound statement block
	}

	if p.checkToken(lexer.OpenBraceToken) {
		namespaceDefinition.CompoundStatementOrSemicolon = p.parseCompoundStatement(namespaceDefinition)
	} else {
		t := &ast.TokenNode{Token: p.eatSemicolonOrAbortStatement()}
		namespaceDefinition.CompoundStatementOrSemicolon = t
	}

	return namespaceDefinition

}

func (p *Parser) parseNamespaceUseDeclaration(parentNode ast.Node) ast.Node {
	namespaceUseDeclaration := &ast.NamespaceUseDeclaration{}
	namespaceUseDeclaration.P = parentNode
	namespaceUseDeclaration.UseKeyword = p.eat1(lexer.UseKeyword)
	namespaceUseDeclaration.FunctionOrConst = p.eatOptional(lexer.FunctionKeyword, lexer.ConstKeyword)
	namespaceUseDeclaration.UseClauses = p.parseNamespaceUseClauseList(namespaceUseDeclaration)
	namespaceUseDeclaration.Semicolon = p.eatSemicolonOrAbortStatement()
	return namespaceUseDeclaration
}

func (p *Parser) parseEmptyStatement(parentNode ast.Node) ast.Node {
	emptyStatement := &ast.EmptyStatement{}
	emptyStatement.P = parentNode
	emptyStatement.Semicolon = p.eat1(lexer.SemicolonToken)
	return emptyStatement
}

func (p *Parser) parseTraitDeclaration(parentNode ast.Node) ast.Node {
	traitDeclaration := &ast.TraitDeclaration{}
	traitDeclaration.P = parentNode

	traitDeclaration.TraitKeyword = p.eat1(lexer.TraitKeyword)
	traitDeclaration.Name = p.eat1(lexer.Name)

	traitDeclaration.TraitMembers = p.parseTraitMembers(traitDeclaration)

	return traitDeclaration
}

func (p *Parser) parseGlobalDeclaration(parentNode ast.Node) ast.Node {
	globalDeclaration := &ast.GlobalDeclaration{}
	globalDeclaration.P = parentNode

	globalDeclaration.GlobalKeyword = p.eat1(lexer.GlobalKeyword)
	variableNameList := &ast.VariableNameList{}
	globalDeclaration.VariableNameList = p.parseDelimitedList(
		variableNameList,
		lexer.CommaToken,
		p.isVariableNameStartFn(),
		p.parseSimpleVariableFn(),
		globalDeclaration,
		false,
	)

	globalDeclaration.Semicolon = p.eatSemicolonOrAbortStatement()
	return globalDeclaration
}

func (p *Parser) parseConstDeclaration(parentNode ast.Node) ast.Node {
	constDeclaration := &ast.ConstDeclaration{}
	constDeclaration.P = parentNode
	constDeclaration.ConstKeyword = p.eat1(lexer.ConstKeyword)
	constDeclaration.ConstElements = p.parseConstElements(constDeclaration)
	constDeclaration.Semicolon = p.eatSemicolonOrAbortStatement()
	return constDeclaration
}

func (p *Parser) parseFunctionStaticDeclaration(parentNode ast.Node) ast.Node {
	functionStaticDeclaration := &ast.FunctionStaticDeclaration{}
	functionStaticDeclaration.P = parentNode

	functionStaticDeclaration.StaticKeyword = p.eat1(lexer.StaticKeyword)
	staticVariableNameList := &ast.StaticVariableNameList{}
	functionStaticDeclaration.StaticVariableNameList = p.parseDelimitedList(
		staticVariableNameList,
		lexer.CommaToken,
		func(token *lexer.Token) bool {
			return token.Kind == lexer.VariableName
		},
		p.parseStaticVariableDeclarationFn(),
		functionStaticDeclaration,
		false,
	)
	functionStaticDeclaration.Semicolon = p.eatSemicolonOrAbortStatement()
	return functionStaticDeclaration
}

func (p *Parser) parseElseIfClause(parentNode *ast.IfStatementNode) ast.Node {
	elseIfClause := &ast.ElseIfClauseNode{}
	elseIfClause.P = parentNode
	elseIfClause.ElseIfKeyword = p.eat1(lexer.ElseIfKeyword)
	elseIfClause.OpenParen = p.eat1(lexer.OpenParenToken)
	elseIfClause.Expression = p.parseExpression(elseIfClause, false)
	elseIfClause.CloseParen = p.eat1(lexer.CloseParenToken)
	if p.checkToken(lexer.ColonToken) {
		elseIfClause.Colon = p.eat1(lexer.ColonToken)
		elseIfClause.Statements = p.parseList(elseIfClause, IfClause2Elements)
	} else {
		elseIfClause.Statements = p.parseStatement(elseIfClause)
	}
	return elseIfClause

}

func (p *Parser) parseElseClause(parentNode *ast.IfStatementNode) ast.Node {
	elseClause := &ast.ElseClauseNode{}
	elseClause.P = parentNode
	elseClause.ElseKeyword = p.eat1(lexer.ElseKeyword)
	if p.checkToken(lexer.ColonToken) {
		elseClause.Colon = p.eat1(lexer.ColonToken)
		elseClause.Statements = p.parseList(elseClause, IfClause2Elements)
	} else {
		elseClause.Statements = p.parseStatement(elseClause)
	}
	return elseClause

}

func (p *Parser) parseScopedPropertyAccessExpression(expression ast.Node) ast.Node {
	scopedPropertyAccessExpression := &ast.ScopedPropertyAccessExpression{}
	scopedPropertyAccessExpression.P = expression.Parent()
	expression.SetParent(scopedPropertyAccessExpression)

	scopedPropertyAccessExpression.ScopeResolutionQualifier = expression // TODO ensure always a Node
	scopedPropertyAccessExpression.DoubleColon = p.eat1(lexer.ColonColonToken)
	scopedPropertyAccessExpression.MemberName = p.parseMemberName(scopedPropertyAccessExpression)
	return scopedPropertyAccessExpression
}

func (p *Parser) parseSubscriptExpression(expression ast.Node) ast.Node {
	subscriptExpression := &ast.SubscriptExpression{}
	subscriptExpression.P = expression.Parent()
	expression.SetParent(subscriptExpression)

	subscriptExpression.PostfixExpression = expression
	subscriptExpression.OpenBracketOrBrace = p.eat(lexer.OpenBracketToken, lexer.OpenBraceToken)
	if p.isExpressionStart(p.token) {
		subscriptExpression.AccessExpression = p.parseExpression(subscriptExpression, false)
	}
	if subscriptExpression.OpenBracketOrBrace.Kind == lexer.OpenBraceToken {
		subscriptExpression.CloseBracketOrBrace = p.eat1(lexer.CloseBraceToken)
	} else {
		subscriptExpression.CloseBracketOrBrace = p.eat1(lexer.CloseBracketToken)
	}

	return subscriptExpression
}

func (p *Parser) parseNumericLiteralExpression(parentNode ast.Node) ast.Node {
	numericLiteral := &ast.NumericLiteral{}
	numericLiteral.P = parentNode
	numericLiteral.Children = p.token
	p.advanceToken()
	return numericLiteral
}

func (p *Parser) parseStringLiteralExpression(parentNode ast.Node) ast.Node {
	// TODO validate input token
	expression := &ast.StringLiteral{}
	expression.P = parentNode
	expression.Children = p.token
	p.advanceToken()
	return expression
}

func (p *Parser) parseAnonymousFunctionCreationExpression(parentNode ast.Node) ast.Node {
	anonymousFunctionCreationExpression := &ast.AnonymousFunctionCreationExpression{}
	anonymousFunctionCreationExpression.P = parentNode

	anonymousFunctionCreationExpression.StaticModifier = p.eatOptional1(lexer.StaticKeyword)
	p.parseFunctionType(anonymousFunctionCreationExpression, false, true)

	return anonymousFunctionCreationExpression

}
func (p *Parser) parseMemberAccessExpression(expression ast.Node) ast.Node {
	memberAccessExpression := &ast.MemberAccessExpression{}
	memberAccessExpression.SetParent(expression.Parent())
	expression.SetParent(memberAccessExpression)

	memberAccessExpression.DereferencableExpression = expression
	memberAccessExpression.ArrowToken = p.eat1(lexer.ArrowToken)
	memberAccessExpression.MemberName = p.parseMemberName(memberAccessExpression)

	return memberAccessExpression

}
func (p *Parser) parseStringLiteralExpression2(parentNode ast.Node) ast.Node {
	// TODO validate input token
	expression := &ast.StringLiteral{}
	expression.P = parentNode
	expression.StartQuote = p.eat(lexer.SingleQuoteToken, lexer.DoubleQuoteToken, lexer.HeredocStart, lexer.BacktickToken)
	children := make([]ast.Node, 0)
	for {
		switch p.token.Kind {
		case lexer.DollarOpenBraceToken,
			lexer.OpenBraceDollarToken:
			t1 := &ast.TokenNode{Token: p.eat(lexer.DollarOpenBraceToken, lexer.OpenBraceDollarToken)}
			children = append(children, t1)
			if p.token.Kind == lexer.StringVarname {
				children = append(children, p.parseSimpleVariable(expression))
			} else {
				children = append(children, p.parseExpression(expression, false))
			}
			t2 := &ast.TokenNode{Token: p.eat1(lexer.CloseBraceToken)}
			children = append(children, t2)
			continue
		case expression.StartQuote.Kind,
			lexer.EndOfFileToken,
			lexer.HeredocEnd:
			expression.EndQuote = p.eat(expression.StartQuote.Kind, lexer.HeredocEnd)
			expression.Children = children
			return expression
		case lexer.VariableName:
			children = append(children, p.parseTemplateStringExpression(expression))
			continue
		default:
			t := &ast.TokenNode{Token: p.token}
			children = append(children, t)
			p.advanceToken()
			continue
		}
	}

}

func (p *Parser) parseCallExpressionRest(expression ast.Node) ast.Node {
	callExpression := &ast.CallExpression{}
	callExpression.P = expression.Parent()
	expression.SetParent(callExpression)
	callExpression.CallableExpression = expression
	callExpression.OpenParen = p.eat1(lexer.OpenParenToken)
	callExpression.ArgumentExpressionList =
		p.parseArgumentExpressionList(callExpression)
	callExpression.CloseParen = p.eat1(lexer.CloseParenToken)
	return callExpression
}

func (p *Parser) parseArrayCreationExpression(parentNode ast.Node) ast.Node {
	arrayExpression := &ast.ArrayCreationExpression{}
	arrayExpression.P = parentNode

	arrayExpression.ArrayKeyword = p.eatOptional1(lexer.ArrayKeyword)

	if arrayExpression.ArrayKeyword != nil {
		arrayExpression.OpenParenOrBracket = p.eat1(lexer.OpenParenToken)
	} else {
		arrayExpression.OpenParenOrBracket = p.eat1(lexer.OpenBracketToken)
	}

	arrayElementList := &ast.ArrayElementList{}
	arrayExpression.ArrayElements = p.parseArrayElementList(arrayExpression, arrayElementList)

	if arrayExpression.ArrayKeyword != nil {
		arrayExpression.CloseParenOrBracket = p.eat1(lexer.CloseParenToken)
	} else {
		arrayExpression.CloseParenOrBracket = p.eat1(lexer.CloseBracketToken)
	}

	return arrayExpression

}

func (p *Parser) parseEchoExpression(parentNode ast.Node) ast.Node {
	echoExpression := &ast.EchoExpression{}
	echoExpression.P = parentNode
	echoExpression.EchoKeyword = p.eat1(lexer.EchoKeyword)
	echoExpression.Expressions = p.parseExpressionList(echoExpression)
	return echoExpression
}

func (p *Parser) parseListIntrinsicExpression(parentNode ast.Node) ast.Node {
	listExpression := &ast.ListIntrinsicExpression{}
	listExpression.P = parentNode
	listExpression.ListKeyword = p.eat1(lexer.ListKeyword)
	listExpression.OpenParen = p.eat1(lexer.OpenParenToken)
	// TODO - parse loosely as ArrayElementList, and validate parse tree later
	listExpressionList := &ast.ListExpressionList{}
	listExpression.ListElements =
		p.parseArrayElementList(listExpression, listExpressionList)
	listExpression.CloseParen = p.eat1(lexer.CloseParenToken)

	return listExpression

}

func (p *Parser) parseUnsetIntrinsicExpression(parentNode ast.Node) ast.Node {
	unsetExpression := &ast.UnsetIntrinsicExpression{}
	unsetExpression.P = parentNode

	unsetExpression.UnsetKeyword = p.eat1(lexer.UnsetKeyword)
	unsetExpression.OpenParen = p.eat1(lexer.OpenParenToken)
	unsetExpression.Expressions = p.parseExpressionList(unsetExpression)
	unsetExpression.CloseParen = p.eat1(lexer.CloseParenToken)

	return unsetExpression

}

func (p *Parser) parseEvalIntrinsicExpression(parentNode ast.Node) ast.Node {
	evalExpression := &ast.EvalIntrinsicExpression{}
	evalExpression.P = parentNode

	evalExpression.EvalKeyword = p.eat1(lexer.EvalKeyword)
	evalExpression.OpenParen = p.eat1(lexer.OpenParenToken)
	evalExpression.Expression = p.parseExpression(evalExpression, false)
	evalExpression.CloseParen = p.eat1(lexer.CloseParenToken)

	return evalExpression

}

func (p *Parser) parseExitIntrinsicExpression(parentNode ast.Node) ast.Node {
	exitExpression := &ast.ExitIntrinsicExpression{}
	exitExpression.P = parentNode

	exitExpression.ExitOrDieKeyword = p.eat(lexer.ExitKeyword, lexer.DieKeyword)
	if exitExpression.ExitOrDieKeyword != nil {
		// normalize always to ExitKeyWord
		exitExpression.ExitOrDieKeyword.Kind = lexer.ExitKeyword
	}
	exitExpression.OpenParen = p.eatOptional1(lexer.OpenParenToken)
	if exitExpression.OpenParen != nil {
		if p.isExpressionStart(p.token) {
			exitExpression.Expression = p.parseExpression(exitExpression, false)
		}
		exitExpression.CloseParen = p.eat1(lexer.CloseParenToken)
	}

	return exitExpression

}

func (p *Parser) parseIssetIntrinsicExpression(parentNode ast.Node) ast.Node {
	issetExpression := &ast.IssetIntrinsicExpression{}
	issetExpression.P = parentNode

	issetExpression.IssetKeyword = p.eat1(lexer.IsSetKeyword)
	issetExpression.OpenParen = p.eat1(lexer.OpenParenToken)
	issetExpression.Expressions = p.parseExpressionList(issetExpression)
	issetExpression.CloseParen = p.eat1(lexer.CloseParenToken)

	return issetExpression
}

func (p *Parser) parsePrintIntrinsicExpression(parentNode ast.Node) ast.Node {
	printExpression := &ast.PrintIntrinsicExpression{}
	printExpression.P = parentNode

	printExpression.PrintKeyword = p.eat1(lexer.PrintKeyword)
	printExpression.Expression = p.parseExpression(printExpression, false)

	return printExpression
}

func (p *Parser) parseReservedWordExpression(parentNode ast.Node) ast.Node {
	reservedWord := &ast.ReservedWord{}
	reservedWord.P = parentNode
	reservedWord.Children = p.token
	p.advanceToken()
	return reservedWord
}

func (p *Parser) parseParenthesizedExpression(parentNode ast.Node) ast.Node {
	parenthesizedExpression := &ast.ParenthesizedExpression{}
	parenthesizedExpression.P = parentNode
	parenthesizedExpression.OpenParen = p.eat1(lexer.OpenParenToken)
	parenthesizedExpression.Expression = p.parseExpression(parenthesizedExpression, false)
	parenthesizedExpression.CloseParen = p.eat1(lexer.CloseParenToken)
	return parenthesizedExpression
}

func (p *Parser) isListTerminator(context ParseContext) bool {
	tokenKind := p.token.Kind
	if tokenKind == lexer.EndOfFileToken {
		// Being at the end of the file ends all lists.
		return true
	}

	switch context {
	case SourceElements:
		return false

	case InterfaceMembers,
		ClassMembers,
		BlockStatements,
		TraitMembers:
		return tokenKind == lexer.CloseBraceToken
	case SwitchStatementElements:
		return tokenKind == lexer.CloseBraceToken || tokenKind == lexer.EndSwitchKeyword
	case IfClause2Elements:
		return tokenKind == lexer.ElseIfKeyword || tokenKind == lexer.ElseKeyword || tokenKind == lexer.EndIfKeyword

	case WhileStatementElements:
		return tokenKind == lexer.EndWhileKeyword

	case CaseStatementElements:
		return tokenKind == lexer.CaseKeyword ||
			tokenKind == lexer.DefaultKeyword

	case ForStatementElements:
		return tokenKind == lexer.EndForKeyword

	case ForeachStatementElements:
		return tokenKind == lexer.EndForEachKeyword

	case DeclareStatementElements:
		return tokenKind == lexer.EndDeclareKeyword
	}
	// TODO warn about unhandled parse context
	return false
}
func (p *Parser) isValidListElement(context ParseContext, token *lexer.Token) bool {
	// TODO
	switch context {
	case SourceElements,
		BlockStatements,
		IfClause2Elements,
		CaseStatementElements,
		WhileStatementElements,
		ForStatementElements,
		ForeachStatementElements,
		DeclareStatementElements:
		return p.isStatementStart(token)

	case ClassMembers:
		return p.isClassMemberDeclarationStart(token)

	case TraitMembers:
		return p.isTraitMemberDeclarationStart(token)

	case InterfaceMembers:
		return p.isInterfaceMemberDeclarationStart(token)

	case SwitchStatementElements:
		return token.Kind == lexer.CaseKeyword || token.Kind == lexer.DefaultKeyword
	}
	return false
}
func (p *Parser) isStatementStart(token *lexer.Token) bool {
	// https://github.com/php/php-langspec/blob/master/spec/19-grammar.md#statements
	switch token.Kind {
	// Compound Statements
	case lexer.OpenBraceToken,
		//Labeled Statements
		lexer.Name,
		//lexer.CaseKeyword: // TODO update spec
		//lexer.DefaultKeyword:
		// Expression Statements
		lexer.SemicolonToken,
		lexer.IfKeyword,
		lexer.SwitchKeyword,
		// Iteration Statements
		lexer.WhileKeyword,
		lexer.DoKeyword,
		lexer.ForKeyword,
		lexer.ForeachKeyword,
		// Jump Statements
		lexer.GotoKeyword,
		lexer.ContinueKeyword,
		lexer.BreakKeyword,
		lexer.ReturnKeyword,
		lexer.ThrowKeyword,
		// The try Statement
		lexer.TryKeyword,
		// The declare Statement
		lexer.DeclareKeyword,
		// const-declaration
		lexer.ConstKeyword,
		// function-definition
		lexer.FunctionKeyword,
		// class-declaration
		lexer.ClassKeyword,
		lexer.AbstractKeyword,
		lexer.FinalKeyword,
		// interface-declaration
		lexer.InterfaceKeyword,
		// trait-declaration
		lexer.TraitKeyword,
		// namespace-definition
		lexer.NamespaceKeyword,
		// namespace-use-declaration
		lexer.UseKeyword,
		// global-declaration
		lexer.GlobalKeyword,
		// function-static-declaration
		lexer.StaticKeyword,
		lexer.ScriptSectionEndTag:
		return true
	}
	return p.isExpressionStart(token)
}

func (p *Parser) isClassMemberDeclarationStart(token *lexer.Token) bool {
	switch token.Kind {
	// const-modifier
	case lexer.ConstKeyword,

		// visibility-modifier
		lexer.PublicKeyword,
		lexer.ProtectedKeyword,
		lexer.PrivateKeyword,

		// static-modifier
		lexer.StaticKeyword,

		// class-modifier
		lexer.AbstractKeyword,
		lexer.FinalKeyword,

		lexer.VarKeyword,

		lexer.FunctionKeyword,

		lexer.UseKeyword:
		return true

	}

	return false
}

func (p *Parser) isCurrentTokenValidInEnclosingContexts() bool {
	var contextKind ParseContext = SourceElements
	for ; contextKind < Count; contextKind++ {
		if p.isInParseContext(contextKind) {
			if p.isValidListElement(contextKind, p.token) || p.isListTerminator(contextKind) {
				return true
			}
		}
	}
	return false
}

func (p *Parser) isInParseContext(context ParseContext) bool {
	return (p.currentParseContext & (1 << context)) != 0
}
func (p *Parser) isTraitMemberDeclarationStart(token *lexer.Token) bool {
	switch token.Kind {
	// property-declaration
	case lexer.VariableName,
		// modifiers
		lexer.PublicKeyword,
		lexer.ProtectedKeyword,
		lexer.PrivateKeyword,
		lexer.VarKeyword,
		lexer.StaticKeyword,
		lexer.AbstractKeyword,
		lexer.FinalKeyword,

		// method-declaration
		lexer.FunctionKeyword,

		// trait-use-clauses
		lexer.UseKeyword:
		return true
	}
	return false
}

func (p *Parser) isInterfaceMemberDeclarationStart(token *lexer.Token) bool {
	switch token.Kind {
	// visibility-modifier
	case lexer.PublicKeyword,
		lexer.ProtectedKeyword,
		lexer.PrivateKeyword,

		// static-modifier
		lexer.StaticKeyword,

		// class-modifier
		lexer.AbstractKeyword,
		lexer.FinalKeyword,

		lexer.ConstKeyword,

		lexer.FunctionKeyword:
		return true
	}
	return false
}

func (p *Parser) getBinaryOperatorPrecedenceAndAssociativity(token *lexer.Token) (int, ast.Assocciativity) {
	val, ok := ast.OPERATOR_PRECEDENCE_AND_ASSOCIATIVITY[token.Kind]
	if ok {
		return val.Precedence, val.Assocc
	}
	return -1, ast.AssocUnknown
}
func (p *Parser) parseTernaryExpression(leftOperand ast.Node, questionToken *lexer.Token) ast.Node {
	ternaryExpression := &ast.TernaryExpression{}
	ternaryExpression.P = leftOperand.Parent()
	leftOperand.SetParent(ternaryExpression)
	ternaryExpression.Condition = leftOperand
	ternaryExpression.QuestionToken = questionToken
	ternaryExpression.IfExpression = nil
	if p.isExpressionStart(p.token) {
		ternaryExpression.IfExpression = p.parseExpression(ternaryExpression, false)
	}
	ternaryExpression.ColonToken = p.eat1(lexer.ColonToken)
	ternaryExpression.ElseExpression = p.parseBinaryExpressionOrHigher(9, ternaryExpression)
	leftOperand = ternaryExpression
	return leftOperand
}

func (p *Parser) makeBinaryAssignmentExpression(leftOperand ast.Node, operatorToken *lexer.Token, byRefToken *lexer.Token, rightOperand ast.Node, parentNode ast.Node) ast.Node {
	binaryExpression := &ast.AssignmentExpression{}
	binaryExpression.P = parentNode
	leftOperand.SetParent(binaryExpression)
	rightOperand.SetParent(binaryExpression)
	binaryExpression.LeftOperand = leftOperand
	binaryExpression.Operator = operatorToken
	if byRefToken != nil {
		binaryExpression.ByRef = byRefToken
	}
	binaryExpression.RightOperand = rightOperand
	return binaryExpression
}

func (p *Parser) makeBinaryExpression(leftOperand ast.Node, operatorToken *lexer.Token, rightOperand ast.Node, parentNode ast.Node) ast.Node {
	binaryExpression := &ast.BinaryExpression{}
	binaryExpression.P = parentNode
	leftOperand.SetParent(binaryExpression)
	rightOperand.SetParent(binaryExpression)
	binaryExpression.LeftOperand = leftOperand
	binaryExpression.Operator = operatorToken
	binaryExpression.RightOperand = rightOperand
	return binaryExpression
}

func (p *Parser) parseTemplateStringExpression(parentNode *ast.StringLiteral) ast.Node {
	token := p.token
	if token.Kind == lexer.VariableName {
		v := p.parseSimpleVariable(parentNode)
		token = p.token
		if token.Kind == lexer.OpenBracketToken {
			return p.parseTemplateStringSubscriptExpression(v)
		} else if token.Kind == lexer.ArrowToken {
			return p.parseTemplateStringMemberAccessExpression(v)
		} else {
			return v
		}
	}
	return nil
}

func (p *Parser) parseTemplateStringSubscriptExpression(postfixExpression ast.Node) ast.Node {
	subscriptExpression := &ast.SubscriptExpression{}
	subscriptExpression.P = postfixExpression.Parent()
	postfixExpression.SetParent(subscriptExpression)

	subscriptExpression.PostfixExpression = postfixExpression
	subscriptExpression.OpenBracketOrBrace = p.eat1(lexer.OpenBracketToken) // Only [] syntax is supported, not {}
	token := p.token
	if token.Kind == lexer.VariableName {
		subscriptExpression.AccessExpression = p.parseSimpleVariable(subscriptExpression)
	} else if token.Kind == lexer.IntegerLiteralToken {
		subscriptExpression.AccessExpression = p.parseNumericLiteralExpression(subscriptExpression)
	} else if token.Kind == lexer.Name {
		subscriptExpression.AccessExpression = p.parseTemplateStringSubscriptStringLiteral(subscriptExpression)
	} else {
		subscriptExpression.AccessExpression = ast.NewMissingToken(lexer.Expression, token.FullStart, nil)
	}
	subscriptExpression.CloseBracketOrBrace = p.eat1(lexer.CloseBracketToken)
	return subscriptExpression
}

func (p *Parser) parseTemplateStringSubscriptStringLiteral(parentNode *ast.SubscriptExpression) ast.Node {
	expression := &ast.StringLiteral{}
	expression.P = parentNode
	expression.Children = p.eat1(lexer.Name)
	return expression
}

func (p *Parser) parseTemplateStringMemberAccessExpression(expression ast.Node) ast.Node {
	memberAccessExpression := &ast.MemberAccessExpression{}
	memberAccessExpression.P = expression.Parent()
	expression.SetParent(memberAccessExpression)

	memberAccessExpression.DereferencableExpression = expression
	memberAccessExpression.ArrowToken = p.eat1(lexer.ArrowToken)
	t := &ast.TokenNode{Token: p.eat1(lexer.Name)}
	memberAccessExpression.MemberName = t

	return memberAccessExpression
}

func (p *Parser) parseMemberName(parentNode ast.Node) ast.Node {
	token := p.token
	switch token.Kind {
	case lexer.Name:
		p.advanceToken() // TODO all names should be Nodes
		tokNode := &ast.TokenNode{}
		tokNode.Token = token
		return tokNode
	case lexer.VariableName,
		lexer.DollarToken:
		return p.parseSimpleVariable(parentNode) // TODO should be simple-variable
	case lexer.OpenBraceToken:
		return p.parseBracedExpression(parentNode)

	default:
		if lexer.IsNameOrKeywordOrReservedWordTokens(token.Kind) {
			p.advanceToken()
			token.Kind = lexer.Name
			tokNode := &ast.TokenNode{}
			tokNode.Token = token
			return tokNode
		}
	}
	return ast.NewMissingToken(lexer.MemberName, p.token.FullStart, nil)
}

func (p *Parser) parseConstElementFn() ParseElementFn {
	return func(parentNode ast.Node) ast.Node {
		constElement := &ast.ConstElement{}
		constElement.P = parentNode
		constElement.Name = p.token
		p.advanceToken()
		constElement.Name.Kind = lexer.Name // to support keyword names
		constElement.EqualsToken = p.eat1(lexer.EqualsToken)
		// TODO add post-parse rule that checks for invalid assignments
		constElement.Assignment = p.parseExpression(constElement, false)
		return constElement
	}
}
func (p *Parser) isParameterStartFn() ElementStartFn {
	return func(token *lexer.Token) bool {
		switch token.Kind {
		case lexer.DotDotDotToken,
			// qualified-name
			lexer.Name, // http://php.net/manual/en/language.namespaces.rules.php
			lexer.BackslashToken,
			lexer.NamespaceKeyword,

			lexer.AmpersandToken,

			lexer.VariableName:
			return true

			// nullable-type
		case lexer.QuestionToken:
			return true
		}
		// scalar-type
		return p.isTokenMember(token.Kind, p.parameterTypeDeclarationTokens)
	}
}

func (p *Parser) parseParameterFn() ParseElementFn {
	return func(parentNode ast.Node) ast.Node {
		parameter := &ast.Parameter{}
		parameter.P = parentNode
		parameter.QuestionToken = p.eatOptional1(lexer.QuestionToken)
		parameter.TypeDeclaration = p.tryParseParameterTypeDeclaration(parameter)
		parameter.ByRefToken = p.eatOptional1(lexer.AmpersandToken)
		// TODO add post-parse rule that prevents assignment
		// TODO add post-parse rule that requires only last parameter be variadic
		parameter.DotDotDotToken = p.eatOptional1(lexer.DotDotDotToken)
		parameter.VariableName = p.eat1(lexer.VariableName)
		parameter.EqualsToken = p.eatOptional1(lexer.EqualsToken)
		if parameter.EqualsToken != nil {
			// TODO add post-parse rule that checks for invalid assignments
			parameter.Default = p.parseExpression(parameter, false)
		}
		return parameter
	}
}

func (p *Parser) isTokenMember(tok lexer.TokenKind, tokens []lexer.TokenKind) bool {
	for _, t := range tokens {
		if t == tok {
			return true
		}
	}
	return false
}

func (p *Parser) tryParseParameterTypeDeclaration(parentNode *ast.Parameter) ast.Node {
	var parameterTypeDeclaration ast.Node
	tn := &ast.TokenNode{Token: p.eatOptional(p.parameterTypeDeclarationTokens...)}
	parameterTypeDeclaration = tn
	if tn.Token == nil {
		parameterTypeDeclaration = p.parseQualifiedName(parentNode)
	}
	return parameterTypeDeclaration
}

func (p *Parser) parseAnonymousFunctionUseClause(parentNode *ast.AnonymousFunctionCreationExpression) ast.Node {
	anonymousFunctionUseClause := &ast.AnonymousFunctionUseClause{}
	anonymousFunctionUseClause.P = parentNode

	anonymousFunctionUseClause.UseKeyword = p.eatOptional1(lexer.UseKeyword)
	if anonymousFunctionUseClause.UseKeyword == nil {
		return nil
	}
	anonymousFunctionUseClause.OpenParen = p.eat1(lexer.OpenParenToken)
	fnName := func(token *lexer.Token) bool {
		return token.Kind == lexer.AmpersandToken || token.Kind == lexer.VariableName
	}
	useVariableNameList := &ast.UseVariableNameList{}
	anonymousFunctionUseClause.UseVariableNameList = p.parseDelimitedList(
		useVariableNameList,
		lexer.CommaToken,
		fnName,
		func(parentNode ast.Node) ast.Node {
			useVariableName := &ast.UseVariableName{}
			useVariableName.P = parentNode
			useVariableName.ByRef = p.eatOptional1(lexer.AmpersandToken)
			useVariableName.VariableName = p.eat1(lexer.VariableName)
			return useVariableName
		},
		anonymousFunctionUseClause, false)
	anonymousFunctionUseClause.CloseParen = p.eat1(lexer.CloseParenToken)

	return anonymousFunctionUseClause
}

func (p *Parser) parseReturnTypeDeclaration(parentNode ast.FunctionInterface) ast.Node {
	tokNode := &ast.TokenNode{Token: p.eatOptional(p.returnTypeDeclarationTokens...)}
	var returnTypeDeclaration ast.Node = tokNode
	if tokNode.Token == nil {
		returnTypeDeclaration = p.parseQualifiedName(parentNode)
	}

	if returnTypeDeclaration == nil {
		returnTypeDeclaration = ast.NewMissingToken(lexer.ReturnType, p.token.FullStart, nil)
	}
	return returnTypeDeclaration
}

func (p *Parser) parseQualifiedNameFn() func(parentNode ast.Node) ast.Node {
	return func(parentNode ast.Node) ast.Node {
		node := &ast.QualifiedName{}
		node.P = parentNode
		node.RelativeSpecifier = p.parseRelativeSpecifier(node)
		if node.RelativeSpecifier == nil {
			node.GlobalSpecifier = p.eatOptional1(lexer.BackslashToken)
		}
		qualifiedNameParts := &ast.QualifiedNameParts{}
		nameParts := p.parseDelimitedList(
			qualifiedNameParts,
			lexer.BackslashToken,
			func(token *lexer.Token) bool {
				// a\static() <- VALID
				// a\static\b <- INVALID
				// a\function <- INVALID
				// a\true\b <-VALID
				// a\b\true <-VALID
				// a\static::b <-VALID
				// TODO more tests

				if p.lookahead(lexer.BackslashToken) {
					return p.isTokenMember(token.Kind, p.nameOrReservedWordTokens)
				}
				return p.isTokenMember(token.Kind, p.nameOrStaticOrReservedWordTokens)
			},
			func(parentNode ast.Node) ast.Node {
				var name *lexer.Token
				if p.lookahead(lexer.BackslashToken) {
					name = p.eat(p.nameOrReservedWordTokens...)
				} else {
					name = p.eat(p.nameOrStaticOrReservedWordTokens...) // TODO support keyword name
				}
				name.Kind = lexer.Name // bool/true/null/static should not be treated as keywords in this case
				return &ast.TokenNode{Token: name}
			}, node, false)

		if (nameParts == nil || nameParts.Len() == 0) && node.GlobalSpecifier == nil && node.RelativeSpecifier == nil {
			return nil
		}

		if nameParts != nil && nameParts.Len() != 0 {
			node.NameParts = nameParts.Children()
		}
		return node
	}

}

func (p *Parser) parseRelativeSpecifier(parentNode ast.Node) ast.Node {
	node := &ast.RelativeSpecifier{}
	node.P = parentNode
	node.NamespaceKeyword = p.eatOptional1(lexer.NamespaceKeyword)
	if node.NamespaceKeyword != nil {
		node.Backslash = p.eat1(lexer.BackslashToken)
	}
	if node.Backslash != nil {
		return node
	}
	return nil
}

func (p *Parser) parseArrayElementList(listExpression ast.Node, delimited ast.DelimitedList) ast.DelimitedList {
	return p.parseDelimitedList(
		delimited,
		lexer.CommaToken,
		p.isArrayElementStartFn(),
		p.parseArrayElementFn(),
		listExpression,
		true,
	)
}

func (p *Parser) isArrayElementStartFn() ElementStartFn {
	return func(token *lexer.Token) bool {
		return token.Kind == lexer.AmpersandToken || p.isExpressionStart(token)
	}
}

func (p *Parser) parseArrayElementFn() ParseElementFn {
	return func(parentNode ast.Node) ast.Node {
		arrayElement := &ast.ArrayElement{}
		arrayElement.P = parentNode

		if p.checkToken(lexer.AmpersandToken) {
			arrayElement.ByRef = p.eat1(lexer.AmpersandToken)
			arrayElement.ElementValue = p.parseExpression(arrayElement, false)
		} else {
			expression := p.parseExpression(arrayElement, false)
			if p.checkToken(lexer.DoubleArrowToken) {
				arrayElement.ElementKey = expression
				arrayElement.ArrowToken = p.eat1(lexer.DoubleArrowToken)
				arrayElement.ByRef = p.eatOptional1(lexer.AmpersandToken) // TODO not okay for list expressions
				arrayElement.ElementValue = p.parseExpression(arrayElement, false)
			} else {
				arrayElement.ElementValue = expression
			}
		}

		return arrayElement
	}

}

func (p *Parser) parseCatchClause(parentNode *ast.TryStatement) ast.Node {
	catchClause := &ast.CatchClause{}
	catchClause.P = parentNode
	catchClause.Catch = p.eat1(lexer.CatchKeyword)
	catchClause.OpenParen = p.eat1(lexer.OpenParenToken)
	catchClause.QualifiedName = p.parseQualifiedName(catchClause) // TODO generate missing token or error if null
	catchClause.VariableName = p.eat1(lexer.VariableName)
	catchClause.CloseParen = p.eat1(lexer.CloseParenToken)
	catchClause.CompoundStatement = p.parseCompoundStatement(catchClause)
	return catchClause
}

func (p *Parser) parseFinallyClause(parentNode *ast.TryStatement) ast.Node {
	finallyClause := &ast.FinallyClause{}
	finallyClause.P = parentNode
	finallyClause.FinallyToken = p.eat1(lexer.FinallyKeyword)
	finallyClause.CompoundStatement = p.parseCompoundStatement(finallyClause)
	return finallyClause
}

func (p *Parser) parseDeclareDirective(parentNode ast.Node) ast.Node {
	declareDirective := &ast.DeclareDirective{}
	declareDirective.P = parentNode
	declareDirective.Name = p.eat1(lexer.Name)
	declareDirective.Equals = p.eat1(lexer.EqualsToken)
	declareDirective.Literal =
		p.eat(
			lexer.FloatingLiteralToken,
			lexer.IntegerLiteralToken,
			lexer.OctalLiteralToken,
			lexer.HexadecimalLiteralToken,
			lexer.BinaryLiteralToken,
			lexer.InvalidOctalLiteralToken,
			lexer.InvalidHexadecimalLiteral,
			lexer.InvalidBinaryLiteral,
			lexer.StringLiteralToken,
		) // TODO simplify

	return declareDirective

}

func (p *Parser) parseNamespaceUseClauseList(parentNode ast.Node) ast.Node {
	namespaceUseClauseList := &ast.NamespaceUseClauseList{}
	return p.parseDelimitedList(
		namespaceUseClauseList,
		lexer.CommaToken,
		func(token *lexer.Token) bool {
			return p.isQualifiedNameStart(token) || token.Kind == lexer.FunctionKeyword || token.Kind == lexer.ConstKeyword
		},
		func(parentNode ast.Node) ast.Node {
			namespaceUseClause := &ast.NamespaceUseClause{}
			namespaceUseClause.P = parentNode
			namespaceUseClause.NamespaceName = p.parseQualifiedName(namespaceUseClause)
			if p.checkToken(lexer.AsKeyword) {
				namespaceUseClause.NamespaceAliasingClause = p.parseNamespaceAliasingClause(namespaceUseClause)
			} else if p.checkToken(lexer.OpenBraceToken) {
				namespaceUseClause.OpenBrace = p.eat1(lexer.OpenBraceToken)
				namespaceUseClause.GroupClauses = p.parseNamespaceUseGroupClauseList(namespaceUseClause)
				namespaceUseClause.CloseBrace = p.eat1(lexer.CloseBraceToken)
			}
			return namespaceUseClause
		},
		parentNode,
		false,
	)
}

func (p *Parser) parseInterfaceBaseClause(parentNode *ast.InterfaceDeclaration) ast.Node {
	interfaceBaseClause := &ast.InterfaceBaseClause{}
	interfaceBaseClause.P = parentNode

	interfaceBaseClause.ExtendsKeyword = p.eatOptional1(lexer.ExtendsKeyword)
	if interfaceBaseClause.ExtendsKeyword != nil {
		interfaceBaseClause.InterfaceNameList = p.parseQualifiedNameList(interfaceBaseClause)
	} else {
		return nil
	}

	return interfaceBaseClause
}

func (p *Parser) parseTraitSelectAndAliasClauseList(parentNode *ast.TraitUseClause) ast.Node {
	traitSelectOrAliasClauseList := &ast.TraitSelectOrAliasClauseList{}
	return p.parseDelimitedList(
		traitSelectOrAliasClauseList,
		lexer.SemicolonToken,
		p.isQualifiedNameStartFn(),
		p.parseTraitSelectOrAliasClauseFn(),
		parentNode,
		false,
	)
}

func (p *Parser) parseInterfaceMembers(parentNode *ast.InterfaceDeclaration) ast.Node {
	interfaceMembers := &ast.InterfaceMembers{}
	interfaceMembers.OpenBrace = p.eat1(lexer.OpenBraceToken)
	interfaceMembers.InterfaceMemberDeclarations = p.parseList(interfaceMembers, InterfaceMembers)
	interfaceMembers.CloseBrace = p.eat1(lexer.CloseBraceToken)
	interfaceMembers.P = parentNode
	return interfaceMembers
}

func (p *Parser) parseTraitMembers(parentNode *ast.TraitDeclaration) ast.Node {
	traitMembers := &ast.TraitMembers{}
	traitMembers.P = parentNode
	traitMembers.OpenBrace = p.eat1(lexer.OpenBraceToken)
	traitMembers.TraitMemberDeclarations = p.parseList(traitMembers, TraitMembers)
	traitMembers.CloseBrace = p.eat1(lexer.CloseBraceToken)
	return traitMembers

}

func (p *Parser) isVariableNameStartFn() ElementStartFn {
	return func(token *lexer.Token) bool {
		return token.Kind == lexer.VariableName || token.Kind == lexer.DollarToken
	}
}

func (p *Parser) parseStaticVariableDeclarationFn() ParseElementFn {
	return func(parentNode ast.Node) ast.Node {
		staticVariableDeclaration := &ast.StaticVariableDeclaration{}
		staticVariableDeclaration.P = parentNode
		staticVariableDeclaration.VariableName = p.eat1(lexer.VariableName)
		staticVariableDeclaration.EqualsToken = p.eatOptional1(lexer.EqualsToken)
		if staticVariableDeclaration.EqualsToken != nil {
			// TODO add post-parse rule that checks for invalid assignments
			staticVariableDeclaration.Assignment = p.parseExpression(staticVariableDeclaration, false)
		}
		return staticVariableDeclaration
	}
}

func (p *Parser) isQualifiedNameStart(token *lexer.Token) bool {
	return (p.isQualifiedNameStartFn())(token)
}

func (p *Parser) isQualifiedNameStartFn() func(*lexer.Token) bool {
	return func(token *lexer.Token) bool {
		switch token.Kind {
		case lexer.BackslashToken,
			lexer.NamespaceKeyword,
			lexer.Name:
			return true
		}
		return false
	}
}

func (p *Parser) parseNamespaceAliasingClause(parentNode ast.Node) ast.Node {
	namespaceAliasingClause := &ast.NamespaceAliasingClause{}
	namespaceAliasingClause.P = parentNode
	namespaceAliasingClause.AsKeyword = p.eat1(lexer.AsKeyword)
	namespaceAliasingClause.Name = p.eat1(lexer.Name)
	return namespaceAliasingClause
}

func (p *Parser) parseNamespaceUseGroupClauseList(parentNode *ast.NamespaceUseClause) ast.Node {
	namespaceUseGroupClauseList := &ast.NamespaceUseGroupClauseList{}
	return p.parseDelimitedList(
		namespaceUseGroupClauseList,
		lexer.CommaToken,
		func(token *lexer.Token) bool {
			return p.isQualifiedNameStart(token) || token.Kind == lexer.FunctionKeyword || token.Kind == lexer.ConstKeyword
		},
		func(parentNode ast.Node) ast.Node {
			namespaceUseGroupClause := &ast.NamespaceUseGroupClause{}
			namespaceUseGroupClause.P = parentNode

			namespaceUseGroupClause.FunctionOrConst = p.eatOptional(lexer.FunctionKeyword, lexer.ConstKeyword)
			namespaceUseGroupClause.NamespaceName = p.parseQualifiedName(namespaceUseGroupClause)
			if p.checkToken(lexer.AsKeyword) {
				namespaceUseGroupClause.NamespaceAliasingClause = p.parseNamespaceAliasingClause(namespaceUseGroupClause)
			}

			return namespaceUseGroupClause
		},
		parentNode,
		false,
	)
}

func (p *Parser) parseTraitSelectOrAliasClauseFn() ParseElementFn {
	return func(parentNode ast.Node) ast.Node {
		traitSelectAndAliasClause := &ast.TraitSelectOrAliasClause{}
		traitSelectAndAliasClause.P = parentNode
		traitSelectAndAliasClause.Name = // TODO update spec
			p.parseQualifiedNameOrScopedPropertyAccessExpression(traitSelectAndAliasClause)

		traitSelectAndAliasClause.AsOrInsteadOfKeyword = p.eat(lexer.AsKeyword, lexer.InsteadOfKeyword)
		traitSelectAndAliasClause.Modifiers = p.parseModifiers() // TODO accept all modifiers, verify later

		traitSelectAndAliasClause.TargetName =
			p.parseQualifiedNameOrScopedPropertyAccessExpression(traitSelectAndAliasClause)

		// TODO errors for insteadof/as
		return traitSelectAndAliasClause
	}
}

func (p *Parser) parseQualifiedNameOrScopedPropertyAccessExpression(parentNode *ast.TraitSelectOrAliasClause) ast.Node {
	qualifiedNameOrScopedProperty := p.parseQualifiedName(parentNode)
	if p.token.Kind == lexer.ColonColonToken {
		qualifiedNameOrScopedProperty = p.parseScopedPropertyAccessExpression(qualifiedNameOrScopedProperty)
	}
	return qualifiedNameOrScopedProperty
}

func (p *Parser) isArgumentExpressionStartFn() ElementStartFn {
	return func(token *lexer.Token) bool {
		if token.Kind == lexer.DotDotDotToken {
			return true
		}
		return p.isExpressionStart(token)
	}
}

func (p *Parser) parseArgumentExpressionFn() ParseElementFn {
	return func(parentNode ast.Node) ast.Node {
		argumentExpression := &ast.ArgumentExpression{}
		argumentExpression.P = parentNode
		argumentExpression.ByRefToken = p.eatOptional1(lexer.AmpersandToken)
		argumentExpression.DotDotDotToken = p.eatOptional1(lexer.DotDotDotToken)
		argumentExpression.Expression = p.parseExpression(argumentExpression, false)
		return argumentExpression
	}
}

func (p *Parser) parseArrayElement(parentNode *ast.YieldExpression) ast.Node {
	return (p.parseArrayElementFn())(parentNode)
}
