package parser

import (
	"testing"
	//"encoding/json"
	"bytes"

	"github.com/emilioastarita/gphp/ast"
	"encoding/json"
	diff "github.com/yudai/gojsondiff"
	"path/filepath"
	"io/ioutil"
	"github.com/yudai/gojsondiff/formatter"
	"os"
)

//func TestParser(t *testing.T) {
//	p := Parser{}
//	sourceFile := p.ParseSourceFile(`<?php echo "test";`, "")
//
//	jsonSource, err := json.Marshal(ast.Serialize(&sourceFile))
//	if err != nil {
//		println(err)
//	} else {
//		pretty, _ := prettyPrintJSON(jsonSource)
//		println(string((pretty)))
//	}
//
//}




func TestCases(t *testing.T) {
	suffix := ".tree"
	tokensLen := len(suffix)

	skipFiles := map[string]bool{

	}
	resultFiles, _ := filepath.Glob("cases/*.php" + suffix)

	for _, resultFile := range resultFiles {

		t.Run(resultFile, func(t *testing.T) {

			if _, skipTest := skipFiles[resultFile]; skipTest {
				return
			}

			resultCase, _ := ioutil.ReadFile(resultFile)
			sourceFileName := resultFile[:len(resultFile)-tokensLen]
			sourceCase, _ := ioutil.ReadFile(sourceFileName)

			p := Parser{}
			sourceFile := p.ParseSourceFile(string(sourceCase), "")
			jsonSource, _ := json.Marshal(ast.Serialize(&sourceFile))

			differ := diff.New()
			d, err := differ.Compare(jsonSource, resultCase)

			if err != nil {
				println(err)
				os.Exit(1)
			}



			if d.Modified() {

				var aJson map[string]interface{}
				json.Unmarshal(jsonSource, &aJson)

				config := formatter.AsciiFormatterConfig{
					ShowArrayIndex: true,
					Coloring:       false,
				}
				formatter := formatter.NewAsciiFormatter(aJson, config)
				diffString, _ := formatter.Format(d)
				t.Log("START DIFF")
				t.Error(diffString)
				t.Log("END DIFF")
			}

		})

	}
}




func prettyPrintJSON(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "    ")
	return out.Bytes(), err
}