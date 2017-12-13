package parser

import (
	"testing"
	//"encoding/json"
	"encoding/json"
	"github.com/emilioastarita/gphp/ast"
	diff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
	"io/ioutil"
	"path/filepath"
)

func TestParser(t *testing.T) {
	p := Parser{}
	sourceFile := p.ParseSourceFile(`<?php

public const a = c;`, "")

	jsonSource, err := json.Marshal(ast.Serialize(&sourceFile))
	if err != nil {
		println(err)
	} else {
		pretty, _ := ast.PrettyPrintJSON(jsonSource)
		println(string((pretty)))
	}

}

func TestCases(t *testing.T) {
	suffix := ".tree"
	tokensLen := len(suffix)

	skipFiles := map[string]bool{}
	resultFiles, _ := filepath.Glob("cases/*.php" + suffix)

	for _, resultFile := range resultFiles {

		t.Run(resultFile, func(t *testing.T) {
			//t.Parallel()
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
