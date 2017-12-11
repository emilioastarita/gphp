package main

import (
	"os"
	"fmt"
	"io/ioutil"
	"encoding/json"
	"github.com/emilioastarita/gphp/parser"
	"github.com/emilioastarita/gphp/ast"
	"path/filepath"
	"github.com/emilioastarita/gphp/lexer"
)

func printUsage() {
	fmt.Println("Usage " + os.Args[0] + " scan|parse filename")
}

func main() {
	if len(os.Args) < 3 {
		printUsage()
		return
	}

	if os.Args[1] == "scan" {
		lexerWalk(os.Args[2])
	} else if os.Args[1] == "parse" {
		printAstFromFile(os.Args[2])
	} else {
		printUsage()
	}


}


func printAstFromFile(filename string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("Can't read file:", filename)
		panic(err)
	}
	content := string(data)
	printAst(content)
}



func printAst(content string) {
	p := parser.Parser{}
	sourceFile := p.ParseSourceFile(content, "")

	jsonSource, err := json.Marshal(ast.Serialize(&sourceFile))

	if err != nil {
		println(err)
	} else {
		pretty, _ := ast.PrettyPrintJSON(jsonSource)
		println(string((pretty)))
	}
}



func lexerWalk(filename string) {
	list, _ := filesOfDir(filename)
	println("Reading: ", len(list))
	for _, f := range list {
		getTokens(f)
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

func getTokens(file string) {
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

