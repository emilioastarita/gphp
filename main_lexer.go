package main

import (
	//"fmt"
	"fmt"
	"io/ioutil"
	"lexer"
	"os"
	"path/filepath"
)

func main() {
	file := os.Args[1]
	list, _ := filesOfDir(file)
	println("Reading: ", len(list))
	for _, f := range list {
		parseFile(f)
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

func parseFile(file string) {
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println("Exit opening file", err)
		return
	}
	//fmt.Println("Data", string(dat));
	tokens := lexer.GetTokens(string(dat))
	fmt.Println("File: ", file, "Tokens: ", len(tokens))
	//for key, token := range tokens {
	//	fmt.Println(key, token)
	//}
}
