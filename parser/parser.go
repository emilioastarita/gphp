package parser

import (
	"github.com/emilioastarita/gphp/ast"
	"github.com/emilioastarita/gphp/lexer"
)

type Parser struct {
	stream                            lexer.TokensStream
	token                             lexer.Token
	currentParseContext               ParseContext
	isParsingObjectCreationExpression bool
	nameOrKeywordOrReservedWordTokens []lexer.TokenKind
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

func (p *Parser) ParseSourceFile(source string, uri string) ast.SourceFile {

	p.nameOrKeywordOrReservedWordTokens = lexer.ReserverTokens()

	p.stream.Source(source)
	p.stream.CreateTokens()
	p.reset()
	sourceFile := ast.SourceFile{P: nil, FileContents: source, Uri: uri}
	if p.token.Kind != lexer.EndOfFileToken {
		sourceFile.Add(p.parseInlineHtml(sourceFile))
	}
	list := p.parseList(sourceFile, SourceElements)
	sourceFile.Merge(list)
	return sourceFile
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
	n := ast.InlineHtml{}
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
		return &t
	}
	return nil
}

func (p *Parser) eatOptional(kinds ... lexer.TokenKind) *lexer.Token {
	t := p.token
	for _, kind := range kinds {
		if t.Kind == kind {
			p.advanceToken()
			return &t
		}
	}
	return nil
}

func (p *Parser) parseList(parentNode ast.Node, listParseContext ParseContext) []ast.Node {
	savedParseContext := p.currentParseContext
	p.currentParseContext |= 1 << listParseContext
	parseListElementFn := p.getParseListElementFn(listParseContext)
	var nodes []ast.Node
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

		t := &lexer.Token{Kind: p.token.Kind, FullStart: p.token.FullStart, Start: p.token.FullStart, Missing: true}
		skipped := &ast.SkippedNode{}
		skipped.Token = t
		nodes = append(nodes, skipped)
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
			missingClassMemberDeclaration := ast.MissingMemberDeclaration{}
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

func (p *Parser) parseModifiers() []lexer.Token {
	var modifiers []lexer.Token
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

		if token.Kind == lexer.EndOfFileToken {
			break
		}

		newPrecedence, associativity := p.getBinaryOperatorPrecedenceAndAssociativity(token)
		if prevAssociativity == ast.AssocNone && prevNewPrecedence == newPrecedence {
			break;
		}
		shouldConsumeCurrentOperator := newPrecedence >= precedence
		if associativity != ast.AssocRight {
			shouldConsumeCurrentOperator = newPrecedence > precedence
		}

		if shouldConsumeCurrentOperator == false {
			break
		}
		unaryExpression, isUnaryExpression := leftOperand.(ast.UnaryOpExpression)
		shouldOperatorTakePrecedenceOverUnary := token.Kind == lexer.AsteriskAsteriskToken && isUnaryExpression

		if shouldOperatorTakePrecedenceOverUnary {
			leftOperand = unaryExpression.Operand;
		}
		p.advanceToken()

		var byRefToken *lexer.Token;
		if token.Kind == lexer.EqualsToken {
			byRefToken = p.eatOptional1(lexer.AmpersandToken)
		}

		if token.Kind == lexer.QuestionToken {
			leftOperand = p.parseTernaryExpression(leftOperand, token)
		} else if token.Kind == lexer.EqualsToken {
			leftOperand = p.makeBinaryAssignmentExpression(leftOperand, token, byRefToken, p.parseBinaryExpressionOrHigher(newPrecedence, nil), parentNode)
		} else {
			leftOperand = p.makeBinaryExpression(leftOperand, token, byRefToken, p.parseBinaryExpressionOrHigher(newPrecedence, nil), parentNode)
		}

		if shouldOperatorTakePrecedenceOverUnary {
			leftOperand.SetParent(unaryExpression)
			unaryExpression.Operand = leftOperand
			leftOperand = unaryExpression
		}

		prevNewPrecedence = newPrecedence
		prevAssociativity = associativity
	}
	return leftOperand;
}

func (p *Parser) parseSimpleVariableFn() func(ast.Node) ast.Node {
	return func(parentNode ast.Node) ast.Node {
		token := p.token
		variable := ast.Variable{}
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
			tokNode := ast.TokenNode{}
			tokNode.Token = tokName
			variable.Name = tokNode
		} else {
			t := &lexer.Token{Kind: lexer.VariableName, FullStart: token.FullStart, Start: token.FullStart, Missing: true}
			missing := ast.Missing{}
			missing.Token = t
			variable.Name = missing
		}

		return variable
	}
}

func (p *Parser) parseBracedExpression(parentNode ast.Node) ast.Node {
	bracedExpression := ast.BracedExpression{}
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
		return lexer.IsReserverdWordToken(token.Kind)
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
				t := &lexer.Token{Kind: lexer.Expression, FullStart: token.FullStart, Start: token.FullStart, Missing: true}
				skipped := &ast.SkippedNode{}
				skipped.Token = t
				return skipped
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

		expressionStatement := ast.ExpressionStatement{}
		expressionStatement.P = parentNode
		expressionStatement.Expression = p.parseExpression(expressionStatement, true)
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
	st := ast.IfStatement{}
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
		st.Statements = []ast.Node{p.parseStatement(st)}
	}
	st.ElseIfClauses = nil
	for p.checkToken(lexer.ElseIfKeyword) {
		st.ElseIfClauses = append(st.ElseIfClauses, p.parseElseIfClause(st))
	}

	if p.checkToken(lexer.ElseKeyword) {
		st.ElseClause = p.parseElseClause(st)
	}

	st.EndifKeyword = p.eatOptional1(lexer.EndIfKeyword)
	if st.EndifKeyword != nil {
		st.SemiColon = p.eatSemicolonOrAbortStatement()
	}

	return st
}

func (p *Parser) parseNamedLabelStatement(parentNode ast.Node) ast.Node {
	st := ast.NamedLabelStatement{}
	st.P = parentNode
	st.Name = p.eat1(lexer.Name)
	st.Colon = p.eat1(lexer.ColonToken)
	st.Statement = p.parseStatement(st)
	return st
}

func (p *Parser) parseCompoundStatement(parentNode ast.Node) ast.Node {
	st := ast.CompoundStatement{}
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
		return &token
	}
	t := &lexer.Token{Kind: kind, FullStart: token.FullStart, Start: token.FullStart, Missing: true}
	return t
}

func (p *Parser) parseExpression(parentNode ast.Node, force bool) ast.Node {
	token := p.token
	if token.Kind == lexer.EndOfFileToken {
		t := &lexer.Token{Kind: lexer.Expression, FullStart: token.FullStart, Start: token.FullStart, Missing: true}
		missing := &ast.Missing{}
		missing.P = parentNode
		missing.Token = t
		return missing
	}
	fnExpression := p.parseExpressionFn()
	expression := fnExpression(parentNode)

	// @todo this not make sense
	// if (force && expression)

	return expression
}
func (p *Parser) checkToken(kind lexer.TokenKind) bool {
	return p.token.Kind == kind
}

func (p *Parser) parseUnaryOpExpression(parent ast.Node) ast.Node {
	st := ast.UnaryOpExpression{}
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
			return &token
		}
	}
	t := &lexer.Token{Kind: kinds[0], FullStart: token.FullStart, Start: token.FullStart, Missing: true}
	return t
}

func (p *Parser) parseErrorControlExpression(parent ast.Node) ast.Node {
	errorExpr := ast.ErrorControlExpression{}
	errorExpr.P = parent
	errorExpr.Operator = p.eat1(lexer.AtSymbolToken)
	operand := p.parseUnaryExpressionOrHigher(errorExpr)
	errorExpr.Operand = operand
	return errorExpr
}

func (p *Parser) parsePrefixUpdateExpression(parent ast.Node) ast.Node {
	n := ast.PrefixUpdateExpression{}
	n.P = parent
	n.IncrementOrDecrementOperator = p.eat(lexer.PlusPlusToken, lexer.MinusMinusToken)
	op := p.parsePrimaryExpression(n)
	n.Operand = op
	switch n.Operand.(type) {
	case ast.Missing:
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
	case ast.Variable,
	ast.ParenthesizedExpression,
	ast.QualifiedName,
	ast.CallExpression,
	ast.MemberAccessExpression,
	ast.SubscriptExpression,
	ast.ScopedPropertyAccessExpression,
	ast.StringLiteral,
	ast.ArrayCreationExpression:
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
		case ast.ArrayCreationExpression:
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
		if p.lookahead([]lexer.TokenKind{lexer.ColonColonToken, lexer.OpenParenToken}) {
			return p.parseQualifiedName(parentNode)
		}
		// Could be `static function` anonymous function creation expression, so flow through
	case lexer.FunctionKeyword:
		return p.parseAnonymousFunctionCreationExpression(parentNode)

	case lexer.TrueReservedWord:
	case lexer.FalseReservedWord:
	case lexer.NullReservedWord:
		// handle `true::`, `true(`, `true\`
		if p.lookahead([]lexer.TokenKind{lexer.BackslashToken, lexer.ColonColonToken, lexer.OpenParenToken}) {
			return p.parseQualifiedName(parentNode)
		}
		return p.parseReservedWordExpression(parentNode)
	}

	if lexer.IsReserverdWordToken(token.Kind) {
		return p.parseQualifiedName(parentNode)
	}

	missing := &ast.Missing{}
	missing.P = parentNode
	missing.Token = &lexer.Token{Kind: lexer.Expression, FullStart: token.FullStart, Start: token.FullStart, Missing: true}
	return missing
}
func (p *Parser) parseSimpleVariable(variable ast.Node) ast.Node {
	fn := p.parseSimpleVariableFn()
	return fn(variable)
}
func (p *Parser) isModifier(token lexer.Token) bool {
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
func (p *Parser) parseClassConstDeclaration(parentNode ast.Node, modifiers []lexer.Token) ast.Node {
	classConstDeclaration := ast.ClassConstDeclaration{}
	classConstDeclaration.P = parentNode
	classConstDeclaration.Modifiers = modifiers
	classConstDeclaration.ConstKeyword = p.eat1(lexer.ConstKeyword)
	classConstDeclaration.ConstElements = p.parseConstElements(classConstDeclaration)
	classConstDeclaration.Semicolon = p.eat1(lexer.SemicolonToken)
	return classConstDeclaration
}

func (p *Parser) parseConstElements(parentNode ast.Node) ast.Node {
	panic("Not implemented parseConstElements")
}

func (p *Parser) parseMethodDeclaration(parentNode ast.Node, modifiers []lexer.Token) ast.Node {
	methodDeclaration := ast.MethodDeclaration{}
	methodDeclaration.Modifiers = modifiers
	p.parseFunctionType(methodDeclaration, true)
	methodDeclaration.P = parentNode
	return methodDeclaration
}

func (p *Parser) parseFunctionType(parent ast.MethodDeclaration, b bool) {
	panic("Not implemented parseFunctionType")
}

func (p *Parser) parsePropertyDeclaration(parentNode ast.Node, modifiers []lexer.Token) ast.Node {
	propertyDeclaration := ast.PropertyDeclaration{}
	propertyDeclaration.P = parentNode
	propertyDeclaration.Modifiers = modifiers
	propertyDeclaration.PropertyElements = p.parseExpressionList(propertyDeclaration)
	propertyDeclaration.Semicolon = p.eat1(lexer.SemicolonToken)
	return propertyDeclaration
}

func (p *Parser) parseExpressionList(parentNode ast.Node) ast.Node {
	expressionList := &ast.ExpressionList{}
	return p.parseDelimitedList(expressionList, lexer.CommaToken, p.isExpressionStartFn(), p.parseExpressionFn(), parentNode, false)
}

type ElementStartFn func(*lexer.Token) bool;

type ParseElementFn func(ast.Node) ast.Node;

func (p *Parser) parseDelimitedList(node ast.DelimitedList, delimiter lexer.TokenKind, isElementStartFn ElementStartFn, parseElementFn ParseElementFn, parentNode ast.Node, allowEmptyElements bool) ast.DelimitedList {
	// TODO consider allowing empty delimiter to be more tolerant
	token := p.token;
	for {
		if isElementStartFn(&token) {
			r := parseElementFn(node)
			node.AddNode(r);
		} else if !allowEmptyElements || (allowEmptyElements && !p.checkToken(delimiter)) {
			break
		}
		delimeterToken := p.eatOptional(delimiter);
		if delimeterToken != nil {
			tokNod := ast.TokenNode{Token: delimeterToken}
			node.AddNode(tokNod);

		}
		token = p.token;
		// TODO ERROR CASE - no delimeter, but a param follows
		if delimeterToken == nil {
			break;
		}
	}

	node.SetParent(parentNode);
	if (node.Children() == nil) {
		return nil;
	}
	return node;
}

func (p *Parser) parseTraitUseClause(parentNode ast.Node) ast.Node {
	panic("Not implemented parseTraitUseClause")
}

func (p *Parser) parseCastExpression(parentNode ast.Node) ast.Node {
	castExpression := ast.CastExpression{}
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
	castExpression := ast.CastExpression{}
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
	objectCreationExpression := ast.ObjectCreationExpression{}
	objectCreationExpression.P = parentNode
	objectCreationExpression.NewKeword = p.eat1(lexer.NewKeyword)
	// TODO - add tests for this scenario
	p.isParsingObjectCreationExpression = true
	if r := p.eatOptional1(lexer.ClassKeyword); r != nil {
		tokNode := ast.TokenNode{}
		tokNode.Token = r
		objectCreationExpression.ClassTypeDesignator = tokNode
	} else if r := p.eatOptional1(lexer.StaticKeyword); r != nil {
		tokNode := ast.TokenNode{}
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
	panic("Not implemented parseArgumentExpressionList")
}

func (p *Parser) parseClassBaseClause(parentNode ast.Node) ast.Node {
	classBaseClause := ast.ClassBaseClause{}
	classBaseClause.P = parentNode
	classBaseClause.ExtendsKeyword = p.eatOptional1(lexer.ExtendsKeyword)
	if classBaseClause.ExtendsKeyword == nil {
		return nil
	}
	classBaseClause.BaseClass = p.parseQualifiedName(classBaseClause)
	return classBaseClause
}
func (p *Parser) parseQualifiedName(parentNode ast.Node) ast.Node {
	panic("Not implemented parseQualifiedName")
}

func (p *Parser) parseClassInterfaceClause(parentNode ast.Node) ast.Node {
	classInterfaceClause := ast.ClassInterfaceClause{}
	classInterfaceClause.P = parentNode
	classInterfaceClause.ImplementsKeyword = p.eatOptional1(lexer.ImplementsKeyword)
	if classInterfaceClause.ImplementsKeyword == nil {
		return nil
	}

	classInterfaceClause.InterfaceNameList = p.parseQualifiedNameList(classInterfaceClause)
	return classInterfaceClause
}

func (p *Parser) parseQualifiedNameList(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}

func (p *Parser) parseCloneExpression(parentNode ast.Node) ast.Node {
	cloneExpression := ast.CloneExpression{}
	cloneExpression.P = parentNode
	cloneExpression.CloneKeyword = p.eat1(lexer.CloneKeyword)
	cloneExpression.Expression = p.parseUnaryExpressionOrHigher(cloneExpression)
	return cloneExpression
}

func (p *Parser) parseYieldExpression(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}

func (p *Parser) parseScriptInclusionExpression(parentNode ast.Node) ast.Node {
	scriptInclusionExpression := ast.ScriptInclusionExpression{}
	scriptInclusionExpression.P = parentNode
	scriptInclusionExpression.RequireOrIncludeKeyword = p.eat(
		lexer.RequireKeyword, lexer.RequireOnceKeyword,
		lexer.IncludeKeyword, lexer.IncludeOnceKeyword)
	scriptInclusionExpression.Expression = p.parseExpression(scriptInclusionExpression, false)
	return scriptInclusionExpression
}

func (p *Parser) parseTraitElementFn() func(ast.Node) ast.Node {
	panic("Not implemented")
}

func (p *Parser) parseInterfaceElementFn() func(ast.Node) ast.Node {
	panic("Not implemented")
}

func (p *Parser) parseCaseOrDefaultStatement() func(ast.Node) ast.Node {
	return func(parentNode ast.Node) ast.Node {
		caseStatement := ast.CaseStatement{}
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
	switchStatement := ast.SwitchStatement{}
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
	whileStatement := ast.WhileStatement{}
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
		whileStatement.Statements = []ast.Node{p.parseStatement(whileStatement)}
	}
	return whileStatement
}

func (p *Parser) parseDoStatement(parentNode ast.Node) ast.Node {
	doStatement := ast.DoStatement{}
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
	forStatement := ast.ForStatement{}
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
		forStatement.Statements = []ast.Node{p.parseStatement(forStatement)}
	}
	return forStatement
}

func (p *Parser) parseForeachStatement(parentNode ast.Node) ast.Node {
	foreachStatement := ast.ForeachStatement{}
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
		foreachStatement.Statements = []ast.Node{p.parseStatement(foreachStatement)}
	}
	return foreachStatement
}
func (p *Parser) tryParseForeachKey(parentNode ast.Node) ast.Node {
	if !p.isExpressionStart(p.token) {
		return nil
	}

	startPos := p.stream.Pos
	startToken := p.token
	foreachKey := ast.ForeachKey{}
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
	foreachValue := ast.ForeachValue{}
	foreachValue.P = parentNode
	foreachValue.Ampersand = p.eatOptional1(lexer.AmpersandToken)
	foreachValue.Expression = p.parseExpression(foreachValue, false)
	return foreachValue
}
func (p *Parser) isExpressionStart(token lexer.Token) bool {
	fn := p.isExpressionStartFn()
	return fn(&token)
}

func (p *Parser) parseEmptyIntrinsicExpression(parentNode ast.Node) ast.Node {
	emptyExpression := ast.EmptyIntrinsicExpression{}
	emptyExpression.P = parentNode
	emptyExpression.EmptyKeyword = p.eat1(lexer.EmptyKeyword)
	emptyExpression.OpenParen = p.eat1(lexer.OpenParenToken)
	emptyExpression.Expression = p.parseExpression(emptyExpression, false)
	emptyExpression.CloseParen = p.eat1(lexer.CloseParenToken)
	return emptyExpression
}
func (p *Parser) parseGotoStatement(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseBreakOrContinueStatement(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseReturnStatement(parentNode ast.Node) ast.Node {
	returnStatement := ast.ReturnStatement{}
	returnStatement.P = parentNode
	returnStatement.ReturnKeyword = p.eat1(lexer.ReturnKeyword)
	if p.isExpressionStart(p.token) {
		returnStatement.Expression = p.parseExpression(returnStatement, false)
	}
	returnStatement.Semicolon = p.eatSemicolonOrAbortStatement()
	return returnStatement
}
func (p *Parser) parseThrowStatement(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseTryStatement(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseDeclareStatement(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseFunctionDeclaration(node ast.Node) ast.Node {
	panic("Not implemented")
	//functionNode := ast.FunctionDeclaration{};
	//p.parseFunctionType(functionNode);
	//functionNode.P = parentNode;
	//return functionNode;
}
func (p *Parser) parseClassMembers(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseClassDeclaration(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseInterfaceDeclaration(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseNamespaceDefinition(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseNamespaceUseDeclaration(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseEmptyStatement(parentNode ast.Node) ast.Node {
	emptyStatement := ast.EmptyStatement{}
	emptyStatement.P = parentNode
	emptyStatement.Semicolon = p.eat1(lexer.SemicolonToken)
	return emptyStatement
}
func (p *Parser) parseTraitDeclaration(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseGlobalDeclaration(parentNode ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseConstDeclaration(parentNode ast.Node) ast.Node {
	constDeclaration := ast.ConstDeclaration{}
	constDeclaration.P = parentNode
	constDeclaration.ConstKeyword = p.eat1(lexer.ConstKeyword)
	constDeclaration.ConstElements = p.parseConstElements(constDeclaration)
	constDeclaration.Semicolon = p.eatSemicolonOrAbortStatement()
	return constDeclaration
}
func (p *Parser) parseFunctionStaticDeclaration(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseElseIfClause(statement ast.IfStatement) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseElseClause(statement ast.IfStatement) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseScopedPropertyAccessExpression(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseSubscriptExpression(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseNumericLiteralExpression(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseStringLiteralExpression(parentNode ast.Node) ast.Node {
	// TODO validate input token
	expression := ast.StringLiteral{};
	expression.P = parentNode;
	expression.Children = &p.token; // TODO - merge string types
	p.advanceToken();
	return expression;
}
func (p *Parser) parseAnonymousFunctionCreationExpression(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseMemberAccessExpression(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseStringLiteralExpression2(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseCallExpressionRest(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseArrayCreationExpression(node ast.Node) ast.Node {
	panic("Not implemented")
}

func (p *Parser) parseEchoExpression(parentNode ast.Node) ast.Node {
	echoExpression := ast.EchoExpression{};
	echoExpression.P = parentNode;
	echoExpression.EchoKeyword = p.eat1(lexer.EchoKeyword);
	echoExpression.Expressions = p.parseExpressionList(echoExpression);
	return echoExpression;
}

func (p *Parser) parseListIntrinsicExpression(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseUnsetIntrinsicExpression(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseEvalIntrinsicExpression(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseExitIntrinsicExpression(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseIssetIntrinsicExpression(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parsePrintIntrinsicExpression(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseReservedWordExpression(node ast.Node) ast.Node {
	panic("Not implemented")
}
func (p *Parser) parseParenthesizedExpression(node ast.Node) ast.Node {
	panic("Not implemented")
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
func (p *Parser) isValidListElement(context ParseContext, token lexer.Token) bool {
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
func (p *Parser) isStatementStart(token lexer.Token) bool {
	// https://github.com/php/php-langspec/blob/master/spec/19-grammar.md#statements
	switch token.Kind {
	// Compound Statements
	case lexer.OpenBraceToken,
		//Labeled Statements
		lexer.Name,
		//case lexer.CaseKeyword: // TODO update spec
		//case lexer.DefaultKeyword:
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

func (p *Parser) isClassMemberDeclarationStart(token lexer.Token) bool {
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
	return (p.currentParseContext & (1 << context)) == 0
}
func (p *Parser) isTraitMemberDeclarationStart(token lexer.Token) bool {
	panic("Not implemented")
}
func (p *Parser) isInterfaceMemberDeclarationStart(token lexer.Token) bool {
	panic("Not implemented")
}

func (p *Parser) getBinaryOperatorPrecedenceAndAssociativity(token lexer.Token) (int, ast.Assocciativity) {
	val, ok := ast.OPERATOR_PRECEDENCE_AND_ASSOCIATIVITY[token.Kind]
	if (ok) {
		return val.Precedence, val.Assocc
	}
	return -1, ast.AssocUnknown
}
func (p *Parser) parseTernaryExpression(leftOperand ast.Node, questionToken lexer.Token) ast.Node {
	ternaryExpression := ast.TernaryExpression{};
	ternaryExpression.P = leftOperand.Parent();
	leftOperand.SetParent(ternaryExpression);
	ternaryExpression.Condition = leftOperand;
	ternaryExpression.QuestionToken = &questionToken;
	ternaryExpression.IfExpression = nil
	if p.isExpressionStart(p.token) {
		ternaryExpression.IfExpression = p.parseExpression(ternaryExpression, false)
	}
	ternaryExpression.ColonToken = p.eat1(lexer.ColonToken);
	ternaryExpression.ElseExpression = p.parseBinaryExpressionOrHigher(9, ternaryExpression);
	leftOperand = ternaryExpression;
	return leftOperand;
}

func (p *Parser) makeBinaryAssignmentExpression(leftOperand ast.Node, operatorToken lexer.Token, byRefToken *lexer.Token, rightOperand ast.Node, parentNode ast.Node) ast.Node {
	binaryExpression := ast.AssignmentExpression{}
	binaryExpression.P = parentNode;
	leftOperand.SetParent(binaryExpression);
	rightOperand.SetParent(binaryExpression);
	binaryExpression.LeftOperand = leftOperand;
	binaryExpression.Operator = &operatorToken;
	if byRefToken != nil {
		binaryExpression.ByRef = byRefToken;
	}
	binaryExpression.RightOperand = rightOperand;
	return binaryExpression;
}

func (p *Parser) makeBinaryExpression(leftOperand ast.Node, operatorToken lexer.Token, byRefToken *lexer.Token, rightOperand ast.Node, parentNode ast.Node) ast.Node {
	binaryExpression := ast.BinaryExpression{}
	binaryExpression.P = parentNode
	leftOperand.SetParent(binaryExpression)
	rightOperand.SetParent(binaryExpression)
	binaryExpression.LeftOperand = leftOperand
	binaryExpression.Operator = &operatorToken
	if byRefToken != nil {
		binaryExpression.ByRef = byRefToken
	}
	binaryExpression.RightOperand = rightOperand
	return binaryExpression;
}
