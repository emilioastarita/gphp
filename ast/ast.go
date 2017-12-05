package ast

import "github.com/emilioastarita/gphp/lexer"


// node
type Node interface {
	Parent() *Node
}

type CNode struct {
	Node
	P *Node
}

type SourceFile struct {
	P              *Node
	FileContents   string
	Uri            string
	StatementList  []*Node
	EndOfFileToken lexer.Token
}

func (s *SourceFile) Add(n Node) {
	s.StatementList = append(s.StatementList, &n)
}

func (s *SourceFile) Merge(nodes []*Node) {
	s.StatementList = append(s.StatementList, nodes...)
}

type Missing struct {
	CNode
	Token *lexer.Token
}

type AnonymousFunctionUseClause struct {
	CNode
	UseKeyword          *lexer.Token
	OpenParen           *lexer.Token
	CloseParen          *lexer.Token
	UseVariableNameList *Node
}

type ArrayElement struct {
	CNode
	ByRef        *lexer.Token
	ArrowToken   *lexer.Token
	ElementKey   *Node
	ElementValue *Node
}

type CaseStatement struct {
	CNode
	CaseKeyword            *lexer.Token
	Expression             *Node
	StatementList          []*Node
	DefaultLabelTerminator *lexer.Token
}

type CatchClause struct {
	CNode

	Catch             *lexer.Token
	OpenParen         *lexer.Token
	VariableName      *lexer.Token
	CloseParen        *lexer.Token
	QualifiedName     *Node
	CompoundStatement *Node
}

type ClassConstDeclaration struct {
	CNode
	Modifiers     []*lexer.Token
	ConstKeyword  *lexer.Token
	Semicolon     *lexer.Token
	ConstElements *Node
}

type ClassBaseClause struct {
	CNode
	ExtendsKeyword *lexer.Token
	BaseClass      *Node
}


type ClassInterfaceClause struct {
	CNode
	ImplementsKeyword *lexer.Token
	InterfaceNameList      *Node
}

// statements

type CompoundStatement struct {
	CNode
	OpenBrace  *lexer.Token
	Statements []*Node
	CloseBrace *lexer.Token
}

type IfStatement struct {
	CNode
	IfKeyword     *lexer.Token
	OpenParen     *lexer.Token
	Expression    *Node
	CloseParen    *lexer.Token
	Colon         *lexer.Token
	Statements    []*Node
	ElseIfClauses []*Node
	ElseClause    *Node
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
	Name       *lexer.Token
	Colon      *lexer.Token
	Statement  *Node
}


// expressions


type UnaryOpExpression struct {
	CNode
	Operator   *lexer.Token
	Operand    *Node
}

type ErrorControlExpression struct {
	UnaryOpExpression
}

type PrefixUpdateExpression struct {
	UnaryOpExpression
	IncrementOrDecrementOperator *lexer.Token
	Operand *Variable
}

type Variable struct {
	CNode
	dollar *lexer.Token
	name *Node
	childNames [2]string
}

// Implements Interface
func (n *CNode) Parent() *Node {
	return n.P
}
