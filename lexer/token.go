package lexer

type Token struct {
	Kind      TokenKind
	FullStart int
	Start     int
	Length    int
	Missing   bool
}

type TokenShortForm struct {
	Kind       string `json:"kind"`
	TextLength int    `json:"textLength"`
	Text       string `json:text`
}

func (r Token) getText(text []rune) string {
	s := r.Start
	end := s + r.Length - (r.Start - r.FullStart)
	return string(text[s:end])
}

func (r Token) getShortForm(text []rune) TokenShortForm {
	return TokenShortForm{r.Kind.String(), r.Length - (r.Start - r.FullStart), r.getText(text)}
}
