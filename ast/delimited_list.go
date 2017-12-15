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
	CNode `serialize:"-"`
	ExpressionListChild
}

type ArgumentExpressionList struct {
	CNode `serialize:"-"`
	ExpressionListChild
}

type QualifiedNameList struct {
	CNode `serialize:"-"`
	ExpressionListChild
}

type ConstElementList struct {
	CNode `serialize:"-"`
	ExpressionListChild
}

type ParameterDeclarationList struct {
	CNode `serialize:"-"`
	ExpressionListChild
}

type UseVariableNameList struct {
	CNode `serialize:"-"`
	ExpressionListChild
}

type QualifiedNameParts struct {
	CNode `serialize:"-"`
	ExpressionListChild
}

type ArrayElementList struct {
	CNode `serialize:"-"`
	ExpressionListChild
}

type ListExpressionList struct {
	CNode `serialize:"-"`
	ExpressionListChild
}

type StaticVariableNameList struct {
	CNode `serialize:"-"`
	ExpressionListChild
}

type NamespaceUseClauseList struct {
	CNode `serialize:"-"`
	ExpressionListChild
}

type VariableNameList struct {
	CNode `serialize:"-"`
	ExpressionListChild
}

type NamespaceUseGroupClauseList struct {
	CNode `serialize:"-"`
	ExpressionListChild
}

type TraitSelectOrAliasClauseList struct {
	CNode `serialize:"-"`
	ExpressionListChild
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
