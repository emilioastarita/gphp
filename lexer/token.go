package lexer

type TokenCategory int

const (
	TokenCatNormal TokenCategory = iota
	TokenCatSkipped
	TokenCatMissing
)

type Token struct {
	Kind      TokenKind
	FullStart int
	Start     int
	Length    int
	Cat       TokenCategory
}

type TokenShortForm struct {
	Kind       string `json:"kind"`
	TextLength int    `json:"textLength"`
	Text       string `json:text`
}

type TokenCompareForm struct {
	Kind      string `json:"kind"`
	FullStart int    `json:"fullStart"`
	Start     int    `json:"start"`
	Length    int    `json:"length"`
}

type TokenFullForm struct {
	Token
	Kind       string `json:"kind"`
	TextLength int    `json:"textLength"`
	Text       string `json:text`
}

func (r Token) getText(text []byte) string {
	s := r.Start
	end := s + r.Length - (r.Start - r.FullStart)
	return string(text[s:end])
}

func (r Token) getShortForm(text []byte) TokenShortForm {
	return TokenShortForm{r.Kind.String(), r.Length - (r.Start - r.FullStart), r.getText(text)}
}

func (r Token) getFullForm(text []byte) TokenFullForm {
	t := TokenFullForm{Kind: r.Kind.String(), TextLength: r.Length - (r.Start - r.FullStart), Text: r.getText(text), Token: r}
	return t
}

func (r Token) getFullFormCompare() TokenCompareForm {
	t := TokenCompareForm{Kind: r.Kind.String(), FullStart: r.FullStart, Start: r.Start, Length: r.Length}
	return t
}
