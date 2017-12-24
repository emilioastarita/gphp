package main

import (
	"encoding/json"
	"fmt"
	"github.com/emilioastarita/gphp/ast"
	"github.com/emilioastarita/gphp/lexer"
	"github.com/emilioastarita/gphp/parser"
	diff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func printUsage() {
	fmt.Println("Usage " + os.Args[0] + " [compare] scan|parse filename")
}

func main() {
	largs := len(os.Args)
	if largs < 3 {
		printUsage()
		return
	}
	isCompare := largs == 4
	action := os.Args[1]
	subAction := ""
	filename := os.Args[2]

	if isCompare {
		subAction = os.Args[2]
		filename = os.Args[3]
	}

	if action == "scan" {
		fnWalk(filename, printTokensFromFile)
	} else if action == "parse" {
		fnWalk(filename, printAstFromFile)
	} else if action == "compare" {
		if subAction == "scan" {
			fnWalk(filename, printDiffWithPhpScan)
		} else if subAction == "parse" {
			fnWalk(filename, printDiffWithPhpParser)
		} else {
			printUsage()
		}
	} else {
		printUsage()
	}

}

func printDiffWithPhpScan(filename string) {
	sourceCase, _ := ioutil.ReadFile(filename)

	stream := lexer.TokensStream{}
	stream.Source(string(sourceCase))
	stream.CreateTokens()

	resultCase := make([]lexer.TokenFullForm, 0)

	json.Unmarshal(getMsParserOutput(filename, "scan"), &resultCase)

	differ := diff.New()

	left, _ := json.Marshal(map[string]interface{}{"_": stream.Serialize()})
	right, _ := json.Marshal(map[string]interface{}{"_": resultCase})

	d, err := differ.Compare(left, right)

	if err != nil {
		fmt.Println("Fail in diff:", filename)
		fmt.Println(err)
		return
	}

	if d.Modified() {
		println("Fail: ", filename)
		var aJson map[string]interface{}
		json.Unmarshal(left, &aJson)

		config := formatter.AsciiFormatterConfig{
			ShowArrayIndex: true,
			Coloring:       false,
		}
		formatter := formatter.NewAsciiFormatter(aJson, config)
		diffString, _ := formatter.Format(d)
		fmt.Println("START DIFF")
		fmt.Println(diffString)
		fmt.Println("END DIFF")

	} else {
		println("Ok: ", filename)
	}
}

func printDiffWithPhpParser(filename string) {
	p := parser.Parser{}
	sourceCase, _ := ioutil.ReadFile(filename)
	sourceFile := p.ParseSourceFile(string(sourceCase), "")
	jsonSource, _ := json.Marshal(ast.Serialize(&sourceFile))
	resultCase := getMsParserOutput(filename, "parse")

	differ := diff.New()

	d, err := differ.Compare(jsonSource, resultCase)

	if err != nil {
		fmt.Println("Fail in diff:", filename)
		fmt.Println(err)
		return
	}

	if d.Modified() {
		println("Fail: ", filename)
		var aJson map[string]interface{}
		json.Unmarshal(jsonSource, &aJson)

		config := formatter.AsciiFormatterConfig{
			ShowArrayIndex: true,
			Coloring:       false,
		}
		formatter := formatter.NewAsciiFormatter(aJson, config)
		diffString, _ := formatter.Format(d)
		fmt.Println("START DIFF")
		fmt.Println(diffString)
		fmt.Println("END DIFF")

	} else {
		println("Ok: ", filename)
	}
}

func getMsParserOutput(filename string, action string) []byte {
	cmd := "/usr/bin/php"
	args := []string{"debug.php", action, filename}
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		log.Fatalln("Error exec: ", cmd, err, string(out))
	}
	return out
}

func printAstFromFile(filename string) {
	fmt.Println("AST of :", filename)
	data, err := ioutil.ReadFile(filename)

	if err != nil {
		fmt.Println("Can't read file:", filename)
		panic(err)
	}
	content := string(data)
	printAst(content)
}

func printTokensFromFile(filename string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("Can't read file:", filename)
		panic(err)
	}
	content := string(data)

	stream := lexer.TokensStream{}
	stream.Source(content)
	stream.CreateTokens()
	stream.Debug()

}

func printAst(content string) {
	p := parser.Parser{}
	sourceFile := p.ParseSourceFile(content, "")

	jsonSource, err := json.Marshal(ast.Serialize(&sourceFile))

	if err != nil {
		println(err)
	} else {
		pretty, _ := ast.PrettyPrintJSON(jsonSource)
		fmt.Println(string((pretty)))
	}
}

func lexerWalk(filename string) {
	list, _ := filesOfDir(filename)
	println("Reading: ", len(list))
	for _, f := range list {
		scanTokens(f)
	}
}

func fnWalk(fileOrDir string, fn func(file string)) {
	fi, err := os.Stat(fileOrDir)
	if err != nil {
		fmt.Println(err)
		return
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		list, _ := filesOfDir(fileOrDir)
		for _, file := range list {
			fn(file)
		}
	case mode.IsRegular():
		fn(fileOrDir)
	}
}

func filesOfDir(searchDir string) ([]string, error) {
	fileList := make([]string, 0)
	e := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() && path[len(path)-4:] == ".php" {
			fileList = append(fileList, path)
		}
		return err
	})

	if e != nil {
		panic(e)
	}
	return fileList, nil
}

func scanTokens(file string) {
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println("Exit opening file", err)
		return
	}
	//fmt.Println("Data", string(dat));
	stream := lexer.TokensStream{}
	stream.Source(string(dat))
	stream.CreateTokens()

	fmt.Println("File: ", file, "Tokens: ", len(stream.Tokens))
	//for key, token := range tokens {
	//	fmt.Println(key, token)
	//}
}
