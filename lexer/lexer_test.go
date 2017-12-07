package lexer

import (
	"encoding/json"
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
			tokens := stream.Tokens

			tokensLen := len(tokens)
			var expectedTokens []TokenShortForm
			json.Unmarshal(resultCase, &expectedTokens)

			for idx, expected := range expectedTokens {

				if idx >= tokensLen {
					t.Fatalf("Failed %s | %s: Expected Kind %s has no match. Actual tokens has Length %d", resultFile, sourceFileName, expected.Kind, tokensLen)
					return
				}

				actual := tokens[idx].getShortForm([]rune(string(sourceCase)))

				if expected.Kind != actual.Kind {
					stream.Debug()
					t.Fatalf("Failed %s | %s: Expected Kind %s - Given Kind: %s", resultFile, sourceFileName, expected.Kind, actual.Kind)
					return
				} else if expected.TextLength != actual.TextLength {
					stream.Debug()
					t.Fatalf("Failed %s: Expected Length Kind %s (len %d) - Given Length Kind: %s (len %d)", resultFile, expected.Kind, expected.TextLength, actual.Kind, actual.TextLength)
					return
				}

			}
		})

	}
}
