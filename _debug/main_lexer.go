package _debug

import (
	"fmt"
	"github.com/emilioastarita/gphp/lexer"
	"io/ioutil"
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
	stream := lexer.TokensStream{}
	stream.Source(string(dat))
	stream.CreateTokens()

	fmt.Println("File: ", file, "Tokens: ", len(stream.Tokens))
	//for key, token := range tokens {
	//	fmt.Println(key, token)
	//}
}
