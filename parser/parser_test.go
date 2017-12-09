package parser

import (
	"testing"
	//"encoding/json"
	"bytes"

	"github.com/emilioastarita/gphp/ast"
	"encoding/json"
)

func TestParser(t *testing.T) {
	p := Parser{}
	sourceFile := p.ParseSourceFile(`<?php echo "test";`, "")


	//ast.Serialize(sourceFile)

	jsonSource, err := json.Marshal(ast.Serialize(&sourceFile))
	if err != nil {
		println(err)
	} else {
		pretty, _ := prettyPrintJSON(jsonSource)
		println(string((pretty)))
	}

}


func prettyPrintJSON(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "    ")
	return out.Bytes(), err
}