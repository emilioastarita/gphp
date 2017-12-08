package parser

import (
	"testing"
	"encoding/json"
	"bytes"
)

func TestParser(t *testing.T) {
	p := Parser{}
	sourceFile := p.ParseSourceFile(`<?php echo "test";`, "")
	jsonSource, err := json.Marshal(sourceFile)
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