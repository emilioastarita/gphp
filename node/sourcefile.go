package node

import "github.com/emilioastarita/gphp/lexer"

type sourceFile struct {
	node
	fileContents   string
	uri            string
	statementList  []Node
	endOfFileToken lexer.Token
	childNames     [2]string
}

func NewSourceFile(fileContents string, uri string) *sourceFile {
	return &sourceFile{
		fileContents: fileContents,
		uri:          uri,
		childNames:   [2]string{"statementList", "endOfFileToken"},
	}
}

func (s *sourceFile) Add(n Node) {
	s.statementList = append(s.statementList, n)
}

func (s *sourceFile) Parent() *Node {
	return s.parent
}
