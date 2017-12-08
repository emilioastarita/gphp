package ast

import "github.com/emilioastarita/gphp/lexer"

// node
type Node interface {
	Parent() Node
	SetParent(p Node)
}

type CNode struct {
	Node
	P Node
}

type SourceFile struct {
	CNode
	P              Node
	FileContents   string
	Uri            string
	StatementList  []Node
	EndOfFileToken lexer.Token
}

func (s *SourceFile) Add(n Node) {
	s.StatementList = append(s.StatementList, n)
}

func (s *SourceFile) Merge(nodes []Node) {
	s.StatementList = append(s.StatementList, nodes...)
}

type Missing struct {
	CNode
	Token *lexer.Token
}

type SkippedNode struct {
	CNode
	Token *lexer.Token
}

type TokenNode struct {
	CNode
	Token *lexer.Token
}

type ForeachKey struct {
	CNode
	Expression Node
	Arrow      *lexer.Token
}

type ForeachValue struct {
	CNode
	Expression Node
	Ampersand  *lexer.Token
}

type AnonymousFunctionUseClause struct {
	CNode
	UseKeyword          *lexer.Token
	OpenParen           *lexer.Token
	CloseParen          *lexer.Token
	UseVariableNameList Node
}

type ArrayElement struct {
	CNode
	ByRef        *lexer.Token
	ArrowToken   *lexer.Token
	ElementKey   Node
	ElementValue Node
}

type CaseStatement struct {
	CNode
	CaseKeyword            *lexer.Token
	Expression             Node
	StatementList          []Node
	DefaultLabelTerminator *lexer.Token
}

type ExpressionStatement struct {
	CNode
	Expression Node
	Semicolon  *lexer.Token
}

type EmptyStatement struct {
	CNode
	Semicolon *lexer.Token
}

type ConstDeclaration struct {
	CNode
	ConstKeyword  *lexer.Token
	ConstElements Node
	Semicolon     *lexer.Token
}

type SwitchStatement struct {
	CNode
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
	CNode
	WhileToken *lexer.Token
	OpenParen  *lexer.Token
	Expression Node
	CloseParen *lexer.Token
	Colon      *lexer.Token
	Statements []Node
	EndWhile   *lexer.Token
	Semicolon  *lexer.Token
}

type DoStatement struct {
	CNode
	Do         *lexer.Token
	Statement  Node
	WhileToken *lexer.Token
	OpenParen  *lexer.Token
	Expression Node
	CloseParen *lexer.Token
	Semicolon  *lexer.Token
}

type ForStatement struct {
	CNode
	For            *lexer.Token
	OpenParen      *lexer.Token
	ForInitializer Node

	ExprGroupSemicolon1 *lexer.Token
	ForControl          Node
	ExprGroupSemicolon2 *lexer.Token
	ForEndOfLoop        Node
	CloseParen          *lexer.Token
	Colon               *lexer.Token
	Statements          []Node
	EndFor              *lexer.Token
	EndForSemicolon     *lexer.Token
}

type ForeachStatement struct {
	CNode
	Foreach               *lexer.Token
	ForEachCollectionName Node
	OpenParen             *lexer.Token
	AsKeyword             *lexer.Token
	ForeachKey            Node
	ForeachValue          Node
	CloseParen            *lexer.Token
	Colon                 *lexer.Token
	Statements            []Node
	EndForeach            *lexer.Token
	EndForeachSemicolon   *lexer.Token
}

type CatchClause struct {
	CNode

	Catch             *lexer.Token
	OpenParen         *lexer.Token
	VariableName      *lexer.Token
	CloseParen        *lexer.Token
	QualifiedName     Node
	CompoundStatement Node
}

type ClassConstDeclaration struct {
	CNode
	Modifiers     []lexer.Token
	ConstKeyword  *lexer.Token
	Semicolon     *lexer.Token
	ConstElements Node
}
type MethodDeclaration struct {
	CNode
	Modifiers []lexer.Token
}

type MissingMemberDeclaration struct {
	CNode
	Modifiers []lexer.Token
}

type QualifiedName struct {
	CNode
	GlobalSpecifier   lexer.Token
	RelativeSpecifier Node
}

type PropertyDeclaration struct {
	CNode
	Modifiers        []lexer.Token
	PropertyElements Node
	Semicolon        *lexer.Token
}

type ClassBaseClause struct {
	CNode
	ExtendsKeyword *lexer.Token
	BaseClass      Node
}

type ClassInterfaceClause struct {
	CNode
	ImplementsKeyword *lexer.Token
	InterfaceNameList Node
}

// statements

type CompoundStatement struct {
	CNode
	OpenBrace  *lexer.Token
	Statements []Node
	CloseBrace *lexer.Token
}

type ReturnStatement struct {
	CNode
	ReturnKeyword *lexer.Token
	Expression    Node
	Semicolon     *lexer.Token
}

type IfStatement struct {
	CNode
	IfKeyword     *lexer.Token
	OpenParen     *lexer.Token
	Expression    Node
	CloseParen    *lexer.Token
	Colon         *lexer.Token
	Statements    []Node
	ElseIfClauses []Node
	ElseClause    Node
	EndifKeyword  *lexer.Token
	SemiColon     *lexer.Token
}

type InlineHtml struct {
	CNode
	ScriptSectionEndTag   *lexer.Token
	Text                  *lexer.Token
	ScriptSectionStartTag *lexer.Token
}

type NamedLabelStatement struct {
	CNode
	Name      *lexer.Token
	Colon     *lexer.Token
	Statement Node
}

// expressions

type UnaryOpExpression struct {
	CNode
	Operator *lexer.Token
	Operand  Node
}

type ErrorControlExpression struct {
	UnaryOpExpression
}

type CastExpression struct {
	UnaryOpExpression
	OpenParen  *lexer.Token
	CastType   *lexer.Token
	CloseParen *lexer.Token
	Operand    Node
}

type PrefixUpdateExpression struct {
	UnaryOpExpression
	IncrementOrDecrementOperator *lexer.Token
	Operand                      Node
}

type PostfixUpdateExpression struct {
	CNode
	IncrementOrDecrementOperator *lexer.Token
	Operand                      Node
}

type CloneExpression struct {
	CNode
	CloneKeyword *lexer.Token
	Expression   Node
}

type EmptyIntrinsicExpression struct {
	CNode
	EmptyKeyword *lexer.Token
	OpenParen    *lexer.Token
	CloseParen   *lexer.Token
	Expression   Node
}

type ParenthesizedExpression struct {
	CNode
	OpenParen  *lexer.Token
	CloseParen *lexer.Token
	Expression Node
}

type CallExpression struct {
	CNode
	OpenParen              *lexer.Token
	CloseParen             *lexer.Token
	CallableExpression     Node
	ArgumentExpressionList Node
}

type MemberAccessExpression struct {
	CNode
	ArrowToken               *lexer.Token
	MemberName               *lexer.Token
	DereferencableExpression Node
}

type SubscriptExpression struct {
	CNode
	OpenBracketOrBrace  *lexer.Token
	CloseBracketOrBrace *lexer.Token
	AccessExpression    Node
	PostfixExpression   Node
}

type ScopedPropertyAccessExpression struct {
	CNode
	ScopeResolutionQualifier *lexer.Token
	DoubleColon              *lexer.Token
	MemberName               Node
}

type ArrayCreationExpression struct {
	CNode
	ArrayKeyword        *lexer.Token
	OpenParenOrBracket  *lexer.Token
	CloseParenOrBracket *lexer.Token
	ArrayElements       Node
}

type StringLiteral struct {
	CNode
	StartQuote *lexer.Token
	Children   Node
	EndQuote   *lexer.Token
}

type ScriptInclusionExpression struct {
	CNode
	RequireOrIncludeKeyword *lexer.Token
	Expression              Node
}

type Variable struct {
	CNode
	Dollar *lexer.Token
	Name   Node
}

type ObjectCreationExpression struct {
	CNode
	NewKeword              *lexer.Token
	ClassTypeDesignator    Node
	OpenParen              *lexer.Token
	ArgumentExpressionList Node
	CloseParen             *lexer.Token
	ClassBaseClause        Node
	ClassInterfaceClause   Node
	ClassMembers           Node
}

type BracedExpression struct {
	CNode
	OpenBrace  *lexer.Token
	Expression Node
	CloseBrace *lexer.Token
}

type BinaryExpression struct {
	CNode
	LeftOperand Node
	Operator *lexer.Token
	RightOperand Node
	ByRef *lexer.Token
}

type EchoExpression struct {
	CNode
	EchoKeyword *lexer.Token
	Expressions Node
}

type AssignmentExpression struct {
	BinaryExpression
}
type TernaryExpression struct {
	CNode
	Condition Node
	IfExpression Node
	ElseExpression Node
	QuestionToken *lexer.Token
	ColonToken *lexer.Token
}

// Implements Interface
func (n CNode) Parent() Node {
	return n.P
}
func (n CNode) SetParent(p Node) {
	n.P = p
}


type Assocciativity int

const (
	AssocNone  Assocciativity = iota
	AssocLeft
	AssocRight
	AssocUnknown
)

type AssocPair struct {
	Precedence int
	Assocc Assocciativity
}

var OPERATOR_PRECEDENCE_AND_ASSOCIATIVITY = map[lexer.TokenKind]AssocPair{
	lexer.OrKeyword: AssocPair{6, AssocLeft},

	// logical-exc-OR-expression-2 (L)
	lexer.XorKeyword: AssocPair{7, AssocLeft},

	// logical-AND-expression-2 (L)
	lexer.AndKeyword: AssocPair{8, AssocLeft},

	// simple-assignment-expression (R)
	// TODO byref-assignment-expression
	lexer.EqualsToken: AssocPair{9, AssocRight},

	// compound-assignment-expression (R)
	lexer.AsteriskAsteriskEqualsToken: AssocPair{9, AssocRight},
	lexer.AsteriskEqualsToken: AssocPair{9, AssocRight},
	lexer.SlashEqualsToken: AssocPair{9, AssocRight},
	lexer.PercentEqualsToken: AssocPair{9, AssocRight},
	lexer.PlusEqualsToken: AssocPair{9, AssocRight},
	lexer.MinusEqualsToken: AssocPair{9, AssocRight},
	lexer.DotEqualsToken: AssocPair{9, AssocRight},
	lexer.LessThanLessThanEqualsToken: AssocPair{9, AssocRight},
	lexer.GreaterThanGreaterThanEqualsToken: AssocPair{9, AssocRight},
	lexer.AmpersandEqualsToken: AssocPair{9, AssocRight},
	lexer.CaretEqualsToken: AssocPair{9, AssocRight},
	lexer.BarEqualsToken: AssocPair{9, AssocRight},

	// TODO conditional-expression (L)
	lexer.QuestionToken: AssocPair{10, AssocLeft},
	//            lexer.ColonToken: AssocPair{9, AssocLeft},

	// TODO coalesce-expression (R)
	lexer.QuestionQuestionToken: AssocPair{9, AssocRight},

	//logical-inc-OR-expression-1 (L)
	lexer.BarBarToken: AssocPair{12, AssocLeft},

	// logical-AND-expression-1 (L)
	lexer.AmpersandAmpersandToken: AssocPair{13, AssocLeft},

	// bitwise-inc-OR-expression (L)
	lexer.BarToken: AssocPair{14, AssocLeft},

	// bitwise-exc-OR-expression (L)
	lexer.CaretToken: AssocPair{15, AssocLeft},

	// bitwise-AND-expression (L)
	lexer.AmpersandToken: AssocPair{16, AssocLeft},

	// equality-expression (X)
	lexer.EqualsEqualsToken: AssocPair{17, AssocNone},
	lexer.ExclamationEqualsToken: AssocPair{17, AssocNone},
	lexer.LessThanGreaterThanToken: AssocPair{17, AssocNone},
	lexer.EqualsEqualsEqualsToken: AssocPair{17, AssocNone},
	lexer.ExclamationEqualsEqualsToken: AssocPair{17, AssocNone},

	// relational-expression (X)
	lexer.LessThanToken: AssocPair{18, AssocNone},
	lexer.GreaterThanToken: AssocPair{18, AssocNone},
	lexer.LessThanEqualsToken: AssocPair{18, AssocNone},
	lexer.GreaterThanEqualsToken: AssocPair{18, AssocNone},
	lexer.LessThanEqualsGreaterThanToken: AssocPair{18, AssocNone},

	// shift-expression (L)
	lexer.LessThanLessThanToken: AssocPair{19, AssocLeft},
	lexer.GreaterThanGreaterThanToken: AssocPair{19, AssocLeft},

	// additive-expression (L)
	lexer.PlusToken: AssocPair{20, AssocLeft},
	lexer.MinusToken: AssocPair{20, AssocLeft},
	lexer.DotToken:AssocPair{20, AssocLeft},

	// multiplicative-expression (L)
	lexer.AsteriskToken: AssocPair{21, AssocLeft},
	lexer.SlashToken: AssocPair{21, AssocLeft},
	lexer.PercentToken: AssocPair{21, AssocLeft},

	// instanceof-expression (X)
	lexer.InstanceOfKeyword: AssocPair{22, AssocNone},

	// exponentiation-expression (R)
	lexer.AsteriskAsteriskToken: AssocPair{23, AssocRight},
}