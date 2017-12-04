package lexer

type Token struct {
	Kind      TokenKind
	FullStart int
	start     int
	length    int
	missing   bool
}

type TokenShortForm struct {
	Kind       string `json:"kind"`
	TextLength int    `json:"textLength"`
	Text       string `json:text`
}

func (r Token) getText(text []rune) string {
	s := r.start
	end := s + r.length - (r.start - r.FullStart)
	return string(text[s:end])
}

func (r Token) getShortForm(text []rune) TokenShortForm {
	return TokenShortForm{r.Kind.String(), r.length - (r.start - r.FullStart), r.getText(text)}
}
