package ast

import "github.com/emilioastarita/gphp/lexer"

type FunctionInterface interface {
	SetFunctionKeyword(v *lexer.Token)
	SetByRefToken(v *lexer.Token)
	SetName(name NodeWithToken)
	SetOpenParen(v *lexer.Token)
	SetCloseParen(v *lexer.Token)
	SetParameters(v Node)
	SetColonToken(v *lexer.Token)
	SetQuestionToken(v *lexer.Token)
	SetReturnType(v Node)
	SetCompoundStatementOrSemicolon(v Node)
	GetName() NodeWithToken
	Node
}

type FunctionHeader struct {
	FunctionKeyword *lexer.Token
	ByRefToken      *lexer.Token
	Name            NodeWithToken
	OpenParen       *lexer.Token
	Parameters      Node
	CloseParen      *lexer.Token
}

func (f *FunctionHeader) SetFunctionKeyword(v *lexer.Token) {
	f.FunctionKeyword = v
}

func (f *FunctionHeader) SetByRefToken(v *lexer.Token) {
	f.ByRefToken = v
}

func (f *FunctionHeader) SetName(name NodeWithToken) {
	f.Name = name
}

func (f *FunctionHeader) GetName() NodeWithToken {
	return f.Name
}

func (f *FunctionHeader) SetOpenParen(v *lexer.Token) {
	f.OpenParen = v
}

func (f *FunctionHeader) SetCloseParen(v *lexer.Token) {
	f.CloseParen = v
}

func (f *FunctionHeader) SetParameters(v Node) {
	f.Parameters = v
}

type FunctionUseClause struct {
	AnonymousFunctionUseClause Node
}

func (f *FunctionUseClause) SetParameters(v Node) {
	f.AnonymousFunctionUseClause = v
}

type FunctionReturnType struct {
	ColonToken    *lexer.Token
	QuestionToken *lexer.Token
	ReturnType    Node
}

func (f *FunctionReturnType) SetColonToken(v *lexer.Token) {
	f.ColonToken = v
}

func (f *FunctionReturnType) SetQuestionToken(v *lexer.Token) {
	f.QuestionToken = v
}

func (f *FunctionReturnType) SetReturnType(v Node) {
	f.ReturnType = v
}

type FunctionBody struct {
	CompoundStatementOrSemicolon Node
}

func (f *FunctionBody) SetCompoundStatementOrSemicolon(v Node) {
	f.CompoundStatementOrSemicolon = v
}

type AnonymousFunctionUseClause struct {
	CNode               `serialize:"-"`
	UseKeyword          *lexer.Token
	OpenParen           *lexer.Token
	CloseParen          *lexer.Token
	UseVariableNameList Node
}

type MethodDeclaration struct {
	CNode              `serialize:"-"`
	FunctionHeader     `serialize:"-flat"`
	FunctionReturnType `serialize:"-flat"`
	FunctionBody       `serialize:"-flat"`
	Modifiers          []*lexer.Token
}
