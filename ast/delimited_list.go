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
