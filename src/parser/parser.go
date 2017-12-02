package parser

type Parser struct {
	pos       int
	eofPos    int
	fullStart int
	start     int
	content   []rune
}

func ParseSourceFile(source string, uri string) {

}
