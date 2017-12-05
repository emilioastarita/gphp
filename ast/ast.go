package ast

import "github.com/emilioastarita/gphp/lexer"


// node
type Node interface {}

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
	P *Node
	Token *lexer.Token
}

type AnonymousFunctionUseClause struct {
	P *Node
	UseKeyword          *lexer.Token
	OpenParen           *lexer.Token
	CloseParen          *lexer.Token
	UseVariableNameList *Node
}

type ArrayElement struct {
	P *Node
	ByRef        *lexer.Token
	ArrowToken   *lexer.Token
	ElementKey   *Node
	ElementValue *Node
}

type CaseStatement struct {
	P *Node
	CaseKeyword            *lexer.Token
	Expression             *Node
	StatementList          []*Node
	DefaultLabelTerminator *lexer.Token
}

type CatchClause struct {
	P *Node

	Catch             *lexer.Token
	OpenParen         *lexer.Token
	VariableName      *lexer.Token
	CloseParen        *lexer.Token
	QualifiedName     *Node
	CompoundStatement *Node
}

type ClassConstDeclaration struct {
	P *Node
	Modifiers     []*lexer.Token
	ConstKeyword  *lexer.Token
	Semicolon     *lexer.Token
	ConstElements *Node
}

type ClassBaseClause struct {
	P *Node
	ExtendsKeyword *lexer.Token
	BaseClass      *Node
}


type ClassInterfaceClause struct {
	P *Node
	ImplementsKeyword *lexer.Token
	InterfaceNameList      *Node
}

// statements

type CompoundStatement struct {
	P *Node
	OpenBrace  *lexer.Token
	Statements []*Node
	CloseBrace *lexer.Token
}

type IfStatement struct {
	P *Node
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
	P *Node
	ScriptSectionEndTag   *lexer.Token
	Text                  *lexer.Token
	ScriptSectionStartTag *lexer.Token
}


type NamedLabelStatement struct {
	P *Node
	Name       *lexer.Token
	Colon      *lexer.Token
	Statement  *Node
}


// expressions


type UnaryOpExpression struct {
	P *Node
	Operator   *lexer.Token
	Operand    *Node
}

type ErrorControlExpression struct {
	UnaryOpExpression
}

type PrefixUpdateExpression struct {
	UnaryOpExpression
	incrementOrDecrementOperator *lexer.Token
	operand *Node
}

type Variable struct {
	P *Node
	dollar *lexer.Token
	name *Node
	childNames [2]string
}

// Node Interface
