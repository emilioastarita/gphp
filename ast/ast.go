package ast

import (
	"github.com/emilioastarita/gphp/lexer"
)

// node
type Node interface {
	Parent() Node
	SetParent(p Node)
}

type NodeWithToken interface {
	Node
	SetToken(*lexer.Token)
	GetToken() *lexer.Token
}

type CNode struct {
	P Node `serialize:"-"`
}

type Parameter struct {
	CNode
	QuestionToken   *lexer.Token
	TypeDeclaration Node
	ByRefToken      *lexer.Token
	DotDotDotToken  *lexer.Token
	VariableName    *lexer.Token
	EqualsToken     *lexer.Token
	Default         Node
}

type UseVariableName struct {
	CNode        `serialize:"-"`
	ByRef        *lexer.Token
	VariableName *lexer.Token
}

type SourceFileNode struct {
	CNode          `serialize:"-"`
	P              Node   `serialize:"-"`
	FileContents   string `serialize:"-"`
	Uri            string `serialize:"-"`
	StatementList  []Node
	EndOfFileToken *lexer.Token
}

func (s *SourceFileNode) Add(n Node) {
	s.StatementList = append(s.StatementList, n)
}

func (s *SourceFileNode) Merge(nodes []Node) {
	s.StatementList = append(s.StatementList, nodes...)
}

type ClassMembersNode struct {
	CNode                   `serialize:"-"`
	OpenBrace               *lexer.Token
	ClassMemberDeclarations []Node
	CloseBrace              *lexer.Token
}

type ForeachKey struct {
	CNode      `serialize:"-"`
	Expression Node
	Arrow      *lexer.Token
}

type ForeachValue struct {
	CNode      `serialize:"-"`
	Expression Node
	Ampersand  *lexer.Token
}

type ArrayElement struct {
	CNode        `serialize:"-"`
	ByRef        *lexer.Token
	ArrowToken   *lexer.Token
	ElementKey   Node
	ElementValue Node
}

type ListIntrinsicExpression struct {
	CNode        `serialize:"-"`
	ListKeyword  *lexer.Token
	OpenParen    *lexer.Token
	CloseParen   *lexer.Token
	ListElements DelimitedList
}

type UnsetIntrinsicExpression struct {
	CNode        `serialize:"-"`
	UnsetKeyword *lexer.Token
	OpenParen    *lexer.Token
	CloseParen   *lexer.Token
	Expressions  DelimitedList
}

type EvalIntrinsicExpression struct {
	CNode       `serialize:"-"`
	EvalKeyword *lexer.Token
	Expression  Node
	OpenParen   *lexer.Token
	CloseParen  *lexer.Token
	Expressions DelimitedList
}

type ExitIntrinsicExpression struct {
	CNode            `serialize:"-"`
	ExitOrDieKeyword *lexer.Token
	Expression       Node
	OpenParen        *lexer.Token
	CloseParen       *lexer.Token
	Expressions      DelimitedList
}

type IssetIntrinsicExpression struct {
	CNode        `serialize:"-"`
	IssetKeyword *lexer.Token
	Expression   Node
	OpenParen    *lexer.Token
	CloseParen   *lexer.Token
	Expressions  DelimitedList
}

type PrintIntrinsicExpression struct {
	CNode        `serialize:"-"`
	PrintKeyword *lexer.Token
	Expression   Node
}

type ReservedWord struct {
	CNode    `serialize:"-"`
	Children *lexer.Token
}

type ConstElement struct {
	CNode       `serialize:"-"`
	Name        *lexer.Token
	EqualsToken *lexer.Token
	Assignment  Node
}

type CaseStatement struct {
	CNode                  `serialize:"-"`
	CaseKeyword            *lexer.Token
	Expression             Node
	StatementList          []Node
	DefaultLabelTerminator *lexer.Token
}

type ArgumentExpression struct {
	CNode          `serialize:"-"`
	ByRefToken     *lexer.Token
	DotDotDotToken *lexer.Token
	Expression     Node
}

type GotoStatement struct {
	CNode     `serialize:"-"`
	Goto      *lexer.Token
	Name      *lexer.Token
	Semicolon *lexer.Token
}

type YieldExpression struct {
	CNode                   `serialize:"-"`
	YieldOrYieldFromKeyword *lexer.Token
	ArrayElement            Node
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

type DeclareStatement struct {
	CNode             `serialize:"-"`
	DeclareKeyword    *lexer.Token
	OpenParen         *lexer.Token
	DeclareDirective  Node
	CloseParen        *lexer.Token
	Colon             *lexer.Token
	Statements        []Node `serialize:"-single"`
	EnddeclareKeyword *lexer.Token
	Semicolon         *lexer.Token
}

type InterfaceDeclaration struct {
	CNode               `serialize:"-"`
	InterfaceKeyword    *lexer.Token
	Name                *lexer.Token
	InterfaceBaseClause Node
	InterfaceMembers    Node
}

type NamespaceDefinition struct {
	CNode                        `serialize:"-"`
	NamespaceKeyword             *lexer.Token
	Name                         Node
	CompoundStatementOrSemicolon Node
}

type NamespaceUseDeclaration struct {
	CNode           `serialize:"-"`
	UseKeyword      *lexer.Token
	FunctionOrConst *lexer.Token
	Semicolon       *lexer.Token
	UseClauses      Node
}

type TraitDeclaration struct {
	CNode        `serialize:"-"`
	TraitKeyword *lexer.Token
	Name         *lexer.Token
	TraitMembers Node
}

type GlobalDeclaration struct {
	CNode            `serialize:"-"`
	GlobalKeyword    *lexer.Token
	VariableNameList Node
	Semicolon        *lexer.Token
}

type FunctionStaticDeclaration struct {
	CNode                  `serialize:"-"`
	StaticKeyword          *lexer.Token
	StaticVariableNameList Node
	Semicolon              *lexer.Token
}

type ElseIfClauseNode struct {
	CNode         `serialize:"-"`
	ElseIfKeyword *lexer.Token
	OpenParen     *lexer.Token
	CloseParen    *lexer.Token
	Expression    Node
	Colon         *lexer.Token
	Statements    []Node
}

type ElseClauseNode struct {
	CNode       `serialize:"-"`
	ElseKeyword *lexer.Token
	Colon       *lexer.Token
	Statements  []Node
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

type ConstDeclaration struct {
	CNode         `serialize:"-"`
	ConstKeyword  *lexer.Token
	ConstElements Node
	Semicolon     *lexer.Token
}

type ClassDeclaration struct {
	CNode                   `serialize:"-"`
	AbstractOrFinalModifier *lexer.Token
	ClassKeyword            *lexer.Token
	Name                    *lexer.Token
	ClassBaseClause         Node
	ClassInterfaceClause    Node
	ClassMembers            Node
}

type SwitchStatement struct {
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
	Statements []Node
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
	Statements          []Node `serialize:"-single"`
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
	Statements            []Node
	EndForeach            *lexer.Token
	EndForeachSemicolon   *lexer.Token
}

type CatchClause struct {
	CNode `serialize:"-"`

	Catch             *lexer.Token
	OpenParen         *lexer.Token
	VariableName      *lexer.Token
	CloseParen        *lexer.Token
	QualifiedName     Node
	CompoundStatement Node
}

type ClassConstDeclaration struct {
	CNode         `serialize:"-"`
	Modifiers     []*lexer.Token
	ConstKeyword  *lexer.Token
	Semicolon     *lexer.Token
	ConstElements Node
}

type MissingMemberDeclaration struct {
	CNode     `serialize:"-"`
	Modifiers []*lexer.Token
}

type QualifiedName struct {
	CNode             `serialize:"-"`
	GlobalSpecifier   *lexer.Token
	RelativeSpecifier Node
	NameParts         []Node
}

type PropertyDeclaration struct {
	CNode            `serialize:"-"`
	Modifiers        []*lexer.Token
	PropertyElements Node
	Semicolon        *lexer.Token
}

type RelativeSpecifier struct {
	CNode            `serialize:"-"`
	NamespaceKeyword *lexer.Token
	Backslash        *lexer.Token
}

type ClassBaseClause struct {
	CNode          `serialize:"-"`
	ExtendsKeyword *lexer.Token
	BaseClass      Node
}

type ClassInterfaceClause struct {
	CNode             `serialize:"-"`
	ImplementsKeyword *lexer.Token
	InterfaceNameList Node
}

// statements

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

type IfStatement struct {
	CNode         `serialize:"-"`
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
	CNode                 `serialize:"-"`
	ScriptSectionEndTag   *lexer.Token
	Text                  *lexer.Token
	ScriptSectionStartTag *lexer.Token
}

type NamedLabelStatement struct {
	CNode     `serialize:"-"`
	Name      *lexer.Token
	Colon     *lexer.Token
	Statement Node
}

// expressions

type UnaryOpExpression struct {
	CNode    `serialize:"-"`
	Operator *lexer.Token
	Operand  Node
}

type ErrorControlExpression struct {
	UnaryOpExpression `serialize:"-flat"`
}

type CastExpression struct {
	UnaryOpExpression `serialize:"-flat"`
	OpenParen         *lexer.Token
	CastType          *lexer.Token
	CloseParen        *lexer.Token
	Operand           Node
}

type PrefixUpdateExpression struct {
	UnaryOpExpression            `serialize:"-flat"`
	IncrementOrDecrementOperator *lexer.Token
	Operand                      Node
}

type PostfixUpdateExpression struct {
	CNode                        `serialize:"-"`
	IncrementOrDecrementOperator *lexer.Token
	Operand                      Node
}

type CloneExpression struct {
	CNode        `serialize:"-"`
	CloneKeyword *lexer.Token
	Expression   Node
}

type EmptyIntrinsicExpression struct {
	CNode        `serialize:"-"`
	EmptyKeyword *lexer.Token
	OpenParen    *lexer.Token
	CloseParen   *lexer.Token
	Expression   Node
}

type ParenthesizedExpression struct {
	CNode      `serialize:"-"`
	OpenParen  *lexer.Token
	CloseParen *lexer.Token
	Expression Node
}

type CallExpression struct {
	CNode                  `serialize:"-"`
	OpenParen              *lexer.Token
	CloseParen             *lexer.Token
	CallableExpression     Node
	ArgumentExpressionList Node
}

type MemberAccessExpression struct {
	CNode                    `serialize:"-"`
	ArrowToken               *lexer.Token
	MemberName               Node
	DereferencableExpression Node
}

type SubscriptExpression struct {
	CNode               `serialize:"-"`
	OpenBracketOrBrace  *lexer.Token
	CloseBracketOrBrace *lexer.Token
	AccessExpression    Node
	PostfixExpression   Node
}

type ScopedPropertyAccessExpression struct {
	CNode                    `serialize:"-"`
	ScopeResolutionQualifier Node
	DoubleColon              *lexer.Token
	MemberName               Node
}

type FinallyClause struct {
	CNode             `serialize:"-"`
	FinallyToken      *lexer.Token
	CompoundStatement Node
}

type DeclareDirective struct {
	CNode   `serialize:"-"`
	Name    *lexer.Token
	Equals  *lexer.Token
	Literal *lexer.Token
}

type NamespaceUseClause struct {
	CNode                   `serialize:"-"`
	NamespaceName           Node
	NamespaceAliasingClause Node
	OpenBrace               *lexer.Token
	GroupClauses            Node
	CloseBrace              *lexer.Token
}

type TraitUseClause struct {
	CNode                      `serialize:"-"`
	UseKeyword                 *lexer.Token
	TraitNameList              Node
	SemicolonOrOpenBrace       *lexer.Token
	TraitSelectAndAliasClauses Node
	CloseBrace                 *lexer.Token
}

type InterfaceBaseClause struct {
	CNode             `serialize:"-"`
	ExtendsKeyword    *lexer.Token
	InterfaceNameList Node
}

type InterfaceMembers struct {
	CNode                       `serialize:"-"`
	OpenBrace                   *lexer.Token
	CloseBrace                  *lexer.Token
	InterfaceMemberDeclarations []Node
}

type TraitMembers struct {
	CNode                   `serialize:"-"`
	OpenBrace               *lexer.Token
	CloseBrace              *lexer.Token
	TraitMemberDeclarations []Node
}

type StaticVariableDeclaration struct {
	CNode        `serialize:"-"`
	VariableName *lexer.Token
	EqualsToken  *lexer.Token
	Assignment   Node
}

type NamespaceAliasingClause struct {
	CNode     `serialize:"-"`
	AsKeyword *lexer.Token
	Name      *lexer.Token
}

type NamespaceUseGroupClause struct {
	CNode                   `serialize:"-"`
	FunctionOrConst         *lexer.Token
	NamespaceName           Node
	NamespaceAliasingClause Node
}

type ArrayCreationExpression struct {
	CNode               `serialize:"-"`
	ArrayKeyword        *lexer.Token
	OpenParenOrBracket  *lexer.Token
	CloseParenOrBracket *lexer.Token
	ArrayElements       DelimitedList
}

type TraitSelectOrAliasClause struct {
	CNode                `serialize:"-"`
	Name                 Node
	AsOrInsteadOfKeyword *lexer.Token
	Modifiers            []*lexer.Token
	TargetName           Node
}

type StringLiteral struct {
	CNode      `serialize:"-"`
	StartQuote *lexer.Token
	Children   []Node `serialize:"-single"`
	EndQuote   *lexer.Token
}

type NumericLiteral struct {
	CNode    `serialize:"-"`
	Children *lexer.Token
}

type ScriptInclusionExpression struct {
	CNode                   `serialize:"-"`
	RequireOrIncludeKeyword *lexer.Token
	Expression              Node
}

type Variable struct {
	CNode  `serialize:"-"`
	Dollar *lexer.Token
	Name   Node
}

type ObjectCreationExpression struct {
	CNode                  `serialize:"-"`
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
	CNode      `serialize:"-"`
	OpenBrace  *lexer.Token
	Expression Node
	CloseBrace *lexer.Token
}

type BinaryExpression struct {
	CNode        `serialize:"-"`
	LeftOperand  Node
	Operator     *lexer.Token
	RightOperand Node
}

type EchoExpression struct {
	CNode       `serialize:"-"`
	EchoKeyword *lexer.Token
	Expressions Node
}

type AssignmentExpression struct {
	BinaryExpression `serialize:"-flat"`
	ByRef            *lexer.Token
}
type TernaryExpression struct {
	CNode          `serialize:"-"`
	Condition      Node
	IfExpression   Node
	ElseExpression Node
	QuestionToken  *lexer.Token
	ColonToken     *lexer.Token
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
	AssocNone Assocciativity = iota
	AssocLeft
	AssocRight
	AssocUnknown
)

type AssocPair struct {
	Precedence int
	Assocc     Assocciativity
}

var OPERATOR_PRECEDENCE_AND_ASSOCIATIVITY = map[lexer.TokenKind]AssocPair{
	lexer.OrKeyword: {6, AssocLeft},

	// logical-exc-OR-expression-2 (L)
	lexer.XorKeyword: {7, AssocLeft},

	// logical-AND-expression-2 (L)
	lexer.AndKeyword: {8, AssocLeft},

	// simple-assignment-expression (R)
	// TODO byref-assignment-expression
	lexer.EqualsToken: {9, AssocRight},

	// compound-assignment-expression (R)
	lexer.AsteriskAsteriskEqualsToken:       {9, AssocRight},
	lexer.AsteriskEqualsToken:               {9, AssocRight},
	lexer.SlashEqualsToken:                  {9, AssocRight},
	lexer.PercentEqualsToken:                {9, AssocRight},
	lexer.PlusEqualsToken:                   {9, AssocRight},
	lexer.MinusEqualsToken:                  {9, AssocRight},
	lexer.DotEqualsToken:                    {9, AssocRight},
	lexer.LessThanLessThanEqualsToken:       {9, AssocRight},
	lexer.GreaterThanGreaterThanEqualsToken: {9, AssocRight},
	lexer.AmpersandEqualsToken:              {9, AssocRight},
	lexer.CaretEqualsToken:                  {9, AssocRight},
	lexer.BarEqualsToken:                    {9, AssocRight},

	// TODO conditional-expression (L)
	lexer.QuestionToken: {10, AssocLeft},
	//            lexer.ColonToken: AssocPair{9, AssocLeft},

	// TODO coalesce-expression (R)
	lexer.QuestionQuestionToken: {9, AssocRight},

	//logical-inc-OR-expression-1 (L)
	lexer.BarBarToken: {12, AssocLeft},

	// logical-AND-expression-1 (L)
	lexer.AmpersandAmpersandToken: {13, AssocLeft},

	// bitwise-inc-OR-expression (L)
	lexer.BarToken: {14, AssocLeft},

	// bitwise-exc-OR-expression (L)
	lexer.CaretToken: {15, AssocLeft},

	// bitwise-AND-expression (L)
	lexer.AmpersandToken: {16, AssocLeft},

	// equality-expression (X)
	lexer.EqualsEqualsToken:            {17, AssocNone},
	lexer.ExclamationEqualsToken:       {17, AssocNone},
	lexer.LessThanGreaterThanToken:     {17, AssocNone},
	lexer.EqualsEqualsEqualsToken:      {17, AssocNone},
	lexer.ExclamationEqualsEqualsToken: {17, AssocNone},

	// relational-expression (X)
	lexer.LessThanToken:                  {18, AssocNone},
	lexer.GreaterThanToken:               {18, AssocNone},
	lexer.LessThanEqualsToken:            {18, AssocNone},
	lexer.GreaterThanEqualsToken:         {18, AssocNone},
	lexer.LessThanEqualsGreaterThanToken: {18, AssocNone},

	// shift-expression (L)
	lexer.LessThanLessThanToken:       {19, AssocLeft},
	lexer.GreaterThanGreaterThanToken: {19, AssocLeft},

	// additive-expression (L)
	lexer.PlusToken:  {20, AssocLeft},
	lexer.MinusToken: {20, AssocLeft},
	lexer.DotToken:   {20, AssocLeft},

	// multiplicative-expression (L)
	lexer.AsteriskToken: {21, AssocLeft},
	lexer.SlashToken:    {21, AssocLeft},
	lexer.PercentToken:  {21, AssocLeft},

	// instanceof-expression (X)
	lexer.InstanceOfKeyword: {22, AssocNone},

	// exponentiation-expression (R)
	lexer.AsteriskAsteriskToken: {23, AssocRight},
}
