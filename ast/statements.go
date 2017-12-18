package ast

import "github.com/emilioastarita/gphp/lexer"

type DeclareStatement struct {
	CNode             `serialize:"-"`
	Statements        NodeOrNodeColl
	DeclareKeyword    *lexer.Token
	OpenParen         *lexer.Token
	DeclareDirective  Node
	CloseParen        *lexer.Token
	Colon             *lexer.Token
	EnddeclareKeyword *lexer.Token
	Semicolon         *lexer.Token
}

type CompoundStatementNode struct {
	CNode      `serialize:"-"`
	OpenBrace  *lexer.Token
	Statements []Node
	CloseBrace *lexer.Token
}

type ReturnStatement struct {
	CNode         `serialize:"-"`
	ReturnKeyword *lexer.Token
	Expression    Node
	Semicolon     *lexer.Token
}

type IfStatementNode struct {
	CNode         `serialize:"-"`
	Statements    NodeOrNodeColl
	IfKeyword     *lexer.Token
	OpenParen     *lexer.Token
	Expression    Node
	CloseParen    *lexer.Token
	Colon         *lexer.Token
	ElseIfClauses []Node
	ElseClause    Node
	EndifKeyword  *lexer.Token
	Semicolon     *lexer.Token
}

type NamedLabelStatement struct {
	CNode     `serialize:"-"`
	Name      *lexer.Token
	Colon     *lexer.Token
	Statement Node
}

type CaseStatementNode struct {
	CNode                  `serialize:"-"`
	CaseKeyword            *lexer.Token
	Expression             Node
	StatementList          []Node
	DefaultLabelTerminator *lexer.Token
}

type GotoStatement struct {
	CNode     `serialize:"-"`
	Goto      *lexer.Token
	Name      *lexer.Token
	Semicolon *lexer.Token
}

type BreakOrContinueStatement struct {
	CNode                  `serialize:"-"`
	BreakOrContinueKeyword *lexer.Token
	BreakoutLevel          Node
	Semicolon              *lexer.Token
}

type ExpressionStatement struct {
	CNode      `serialize:"-"`
	Expression []Node `serialize:"-single"`
	Semicolon  *lexer.Token
}

type ThrowStatement struct {
	CNode        `serialize:"-"`
	Expression   Node
	ThrowKeyword *lexer.Token
	Semicolon    *lexer.Token
}

type TryStatement struct {
	CNode             `serialize:"-"`
	TryKeyword        *lexer.Token
	CompoundStatement Node
	CatchClauses      []Node
	FinallyClause     Node
}

type EmptyStatement struct {
	CNode     `serialize:"-"`
	Semicolon *lexer.Token
}

type ElseIfClauseNode struct {
	CNode         `serialize:"-"`
	ElseIfKeyword *lexer.Token
	OpenParen     *lexer.Token
	CloseParen    *lexer.Token
	Expression    Node
	Colon         *lexer.Token
	Statements    NodeOrNodeColl
}

type ElseClauseNode struct {
	CNode `serialize:"-"`

	ElseKeyword *lexer.Token
	Colon       *lexer.Token
	Statements  NodeOrNodeColl
}

type SwitchStatementNode struct {
	CNode          `serialize:"-"`
	SwitchKeyword  *lexer.Token
	OpenParen      *lexer.Token
	Expression     Node
	CloseParen     *lexer.Token
	Colon          *lexer.Token
	OpenBrace      *lexer.Token
	CaseStatements []Node
	CloseBrace     *lexer.Token
	Endswitch      *lexer.Token
	Semicolon      *lexer.Token
}

type WhileStatement struct {
	CNode      `serialize:"-"`
	WhileToken *lexer.Token
	OpenParen  *lexer.Token
	Expression Node
	CloseParen *lexer.Token
	Colon      *lexer.Token
	Statements NodeOrNodeColl
	EndWhile   *lexer.Token
	Semicolon  *lexer.Token
}

type DoStatement struct {
	CNode      `serialize:"-"`
	Do         *lexer.Token
	Statement  Node
	WhileToken *lexer.Token
	OpenParen  *lexer.Token
	Expression Node
	CloseParen *lexer.Token
	Semicolon  *lexer.Token
}

type ForStatement struct {
	CNode          `serialize:"-"`
	For            *lexer.Token
	OpenParen      *lexer.Token
	ForInitializer Node

	ExprGroupSemicolon1 *lexer.Token
	ForControl          Node
	ExprGroupSemicolon2 *lexer.Token
	ForEndOfLoop        Node
	CloseParen          *lexer.Token
	Colon               *lexer.Token
	Statements          NodeOrNodeColl
	EndFor              *lexer.Token
	EndForSemicolon     *lexer.Token
}

type ForeachStatement struct {
	CNode                 `serialize:"-"`
	Foreach               *lexer.Token
	ForEachCollectionName Node
	OpenParen             *lexer.Token
	AsKeyword             *lexer.Token
	ForeachKey            Node
	ForeachValue          Node
	CloseParen            *lexer.Token
	Colon                 *lexer.Token
	Statements            NodeOrNodeColl
	EndForeach            *lexer.Token
	EndForeachSemicolon   *lexer.Token
}
