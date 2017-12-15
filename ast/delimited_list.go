package ast

type DelimitedList interface {
	Node
	AddNode(n Node)
	Children() []Node
}

type ExpressionList struct {
	CNode  `serialize:"-"`
	Childs []Node `serialize:"children"`
}

type ConstElementList struct {
	CNode  `serialize:"-"`
	Childs []Node `serialize:"children"`
}

type ParameterDeclarationList struct {
	CNode  `serialize:"-"`
	Childs []Node `serialize:"children"`
}

type UseVariableNameList struct {
	CNode  `serialize:"-"`
	Childs []Node `serialize:"children"`
}

type QualifiedNameParts struct {
	CNode  `serialize:"-"`
	Childs []Node `serialize:"children"`
}

type ArrayElementList struct {
	CNode  `serialize:"-"`
	Childs []Node `serialize:"children"`
}

type ListExpressionList struct {
	CNode  `serialize:"-"`
	Childs []Node `serialize:"children"`
}

type StaticVariableNameList struct {
	CNode  `serialize:"-"`
	Childs []Node `serialize:"children"`
}

type NamespaceUseClauseList struct {
	CNode  `serialize:"-"`
	Childs []Node `serialize:"children"`
}

type VariableNameList struct {
	CNode  `serialize:"-"`
	Childs []Node `serialize:"children"`
}

type NamespaceUseGroupClauseList struct {
	CNode  `serialize:"-"`
	Childs []Node `serialize:"children"`
}

type TraitSelectOrAliasClauseList struct {
	CNode  `serialize:"-"`
	Childs []Node `serialize:"children"`
}

func (e *ExpressionList) AddNode(node Node) {
	if node == nil {
		return
	}
	e.Childs = append(e.Childs, node)
}

func (e *ExpressionList) Children() []Node {
	return e.Childs
}

func (e *ConstElementList) AddNode(node Node) {
	if node == nil {
		return
	}
	e.Childs = append(e.Childs, node)
}

func (e *ConstElementList) Children() []Node {
	return e.Childs
}

func (e *ParameterDeclarationList) AddNode(node Node) {
	if node == nil {
		return
	}
	e.Childs = append(e.Childs, node)
}

func (e *ParameterDeclarationList) Children() []Node {
	return e.Childs
}

func (e *UseVariableNameList) AddNode(node Node) {
	if node == nil {
		return
	}
	e.Childs = append(e.Childs, node)
}

func (e *UseVariableNameList) Children() []Node {
	return e.Childs
}

func (e *QualifiedNameParts) AddNode(node Node) {
	if node == nil {
		return
	}
	e.Childs = append(e.Childs, node)
}

func (e *QualifiedNameParts) Children() []Node {
	return e.Childs
}

func (e *ArrayElementList) AddNode(node Node) {
	if node == nil {
		return
	}
	e.Childs = append(e.Childs, node)
}

func (e *ArrayElementList) Children() []Node {
	return e.Childs
}

func (e *ListExpressionList) AddNode(node Node) {
	if node == nil {
		return
	}
	e.Childs = append(e.Childs, node)
}

func (e *ListExpressionList) Children() []Node {
	return e.Childs
}

func (e *StaticVariableNameList) AddNode(node Node) {
	if node == nil {
		return
	}
	e.Childs = append(e.Childs, node)
}

func (e *StaticVariableNameList) Children() []Node {
	return e.Childs
}

func (e *NamespaceUseClauseList) AddNode(node Node) {
	if node == nil {
		return
	}
	e.Childs = append(e.Childs, node)
}

func (e *NamespaceUseClauseList) Children() []Node {
	return e.Childs
}

func (e *VariableNameList) AddNode(node Node) {
	if node == nil {
		return
	}
	e.Childs = append(e.Childs, node)
}

func (e *VariableNameList) Children() []Node {
	return e.Childs
}

func (e *NamespaceUseGroupClauseList) AddNode(node Node) {
	if node == nil {
		return
	}
	e.Childs = append(e.Childs, node)
}

func (e *NamespaceUseGroupClauseList) Children() []Node {
	return e.Childs
}

func (e *TraitSelectOrAliasClauseList) AddNode(node Node) {
	if node == nil {
		return
	}
	e.Childs = append(e.Childs, node)
}

func (e *TraitSelectOrAliasClauseList) Children() []Node {
	return e.Childs
}
