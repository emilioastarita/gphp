package ast

type DelimitedList interface {
	Node
	AddNode(n Node)
	Children() []Node
}

type ExpressionListChild struct {
	Child []Node `serialize:"children"`
}

type ExpressionList struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

type ArgumentExpressionList struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

type QualifiedNameList struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

type ConstElementList struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

type ParameterDeclarationList struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

type UseVariableNameList struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

type QualifiedNameParts struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

type ArrayElementList struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

type ListExpressionList struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

type StaticVariableNameList struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

type NamespaceUseClauseList struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

type VariableNameList struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

type NamespaceUseGroupClauseList struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

type TraitSelectOrAliasClauseList struct {
	CNode               `serialize:"-"`
	ExpressionListChild `serialize:"-flat"`
}

func (e *ExpressionListChild) AddNode(node Node) {
	if node == nil {
		return
	}
	e.Child = append(e.Child, node)
}

func (e *ExpressionListChild) Children() []Node {
	return e.Child
}
