package main

import (
	//"fmt"
	"io/ioutil"
	"lexer"
)

func main() {
	dat, _ := ioutil.ReadFile("src/example.php")
	//fmt.Println("Data", string(dat));
	_ = lexer.GetTokens(string(dat))
	//for key, token := range tokens {
	//	fmt.Println("%s", key, token.Kind.String())
	//}
}
