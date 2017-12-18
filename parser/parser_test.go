package parser

import (
	"testing"
	//"encoding/json"
	"encoding/json"
	"github.com/emilioastarita/gphp/ast"
	diff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
	"io/ioutil"
	path "path"
	"path/filepath"
)

func TestParser(t *testing.T) {
	p := Parser{}
	sourceFile := p.ParseSourceFile(`<?php

function foobar ($a, ...$b) {

}`, "")

	jsonSource, err := json.Marshal(ast.Serialize(&sourceFile))
	if err != nil {
		println(err)
	} else {
		pretty, _ := ast.PrettyPrintJSON(jsonSource)
		println(string((pretty)))
	}

}

func TestCases(t *testing.T) {
	postfix := ".tree"
	postfixLen := len(postfix)

	skipFiles := SKIPPED_TESTS
	resultFiles, _ := filepath.Glob("cases/*.php" + postfix)

	for _, resultFile := range resultFiles {

		t.Run(resultFile, func(t *testing.T) {
			//t.Parallel()

			resultCase, _ := ioutil.ReadFile(resultFile)
			sourceFileName := resultFile[:len(resultFile)-postfixLen]

			if _, skipTest := skipFiles[path.Base(sourceFileName)]; skipTest {
				t.Log("Skipped: " + resultFile)
				return
			}

			sourceCase, _ := ioutil.ReadFile(sourceFileName)

			p := Parser{}
			sourceFile := p.ParseSourceFile(string(sourceCase), "")
			jsonSource, _ := json.Marshal(ast.Serialize(&sourceFile))

			differ := diff.New()
			d, err := differ.Compare(jsonSource, resultCase)

			if err != nil {
				t.Log("Log comparing json sources:")
				t.Log(err)
				ast.PrettyPrintJSON(jsonSource)
				t.Error("Test fail.")
				return
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
