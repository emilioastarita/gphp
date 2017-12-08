package parser

import (
	"testing"
	"encoding/json"
	"bytes"
	"github.com/emilioastarita/gphp/ast"
)

func TestParser(t *testing.T) {
	p := Parser{}
	sourceFile := p.ParseSourceFile(`<?php echo "test";`, "")
	jsonSource, err := json.Marshal(ast.Serializable(sourceFile))
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