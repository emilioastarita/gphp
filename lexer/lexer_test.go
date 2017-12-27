package lexer

import (
	"encoding/json"
	diff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func BenchmarkComplex(b *testing.B) {
	data, _ := ioutil.ReadFile("cases/complex.php")
	stream := TokensStream{}

	for n := 0; n < b.N; n++ {
		stream.Source(string(data))
		stream.CreateTokens()
	}
}

func TestEx(t *testing.T) {
	stream := TokensStream{}
	stream.Source(`<?php
(array)
`)
	stream.CreateTokens()
	stream.Debug()
}

func TestCases(t *testing.T) {
	suffix := ".tokens"
	tokensLen := len(suffix)

	skipFiles := map[string]bool{
		"cases/keyword4.php.tokens": true,
		"cases/keyword6.php.tokens": true,
		"cases/keyword7.php.tokens": true,
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
			stream := TokensStream{}
			stream.Source(string(sourceCase))
			stream.CreateTokens()

			var expectedTokens []TokenCompareForm
			json.Unmarshal(resultCase, &expectedTokens)

			differ := diff.New()
			left, _ := json.Marshal(map[string]interface{}{"_": stream.Serialize()})
			right, _ := json.Marshal(map[string]interface{}{"_": expectedTokens})

			d, err := differ.Compare(left, right)

			if err != nil {
				t.Log("Fail in diff:", resultCase)
				t.Error(err)
				return
			}

			if d.Modified() {
				t.Logf("Json modified: %s", resultFile)
				var aJson map[string]interface{}
				json.Unmarshal(left, &aJson)

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
