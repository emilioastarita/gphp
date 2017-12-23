package lexer

import (
	"encoding/json"
	"os"
	"strings"
	"unicode"
)

type LexerState int
type HereDocStatus int

const (
	LexStateHtmlSection LexerState = iota
	LexStateScriptSection
	LexStateScriptSectionParsed
)
const (
	HereDocStateNone HereDocStatus = iota
	HereDocNormal
	HereDocNowDoc
)

type LexerScanner struct {
	state             LexerState
	hereDocStatus     HereDocStatus
	hereDocIdentifier string
	pos               int
	eofPos            int
	fullStart         int
	start             int
	content           []rune
}

type TokensStream struct {
	Tokens []*Token
	Pos    int
	EofPos int

	tokenMem []*Token
	lexer    LexerScanner
}

func (s *TokensStream) Source(content string) {
	s.lexer = LexerScanner{
		LexStateHtmlSection,
		HereDocStateNone,
		"",
		0,
		0,
		0,
		0,
		[]rune(content),
	}
	s.lexer.eofPos = len(s.lexer.content)
}

func (s *TokensStream) CreateTokens() {
	lexer := s.lexer
	var token *Token = &Token{}
	for token.Kind != EndOfFileToken {
		token, s.tokenMem = lexer.scan(nil)
		if token.Kind == -1 {
			s.Tokens = append(s.Tokens, s.tokenMem...)
		} else {
			s.Tokens = append(s.Tokens, token)
			lexer.pos = token.FullStart + token.Length
		}
	}
	s.Pos = 0
	s.EofPos = len(s.Tokens) - 1
}

func (s *TokensStream) ScanNext() *Token {
	if s.Pos >= s.EofPos {
		return s.Tokens[s.EofPos]
	}
	pos := s.Pos
	s.Pos++
	return s.Tokens[pos]
}

func (s *TokensStream) Debug() {
	for _, token := range s.Tokens {
		b, _ := json.MarshalIndent(token.getFullForm([]rune(s.lexer.content)), "", "    ")
		os.Stdout.Write(b)
		println("")
	}
}

func (l *LexerScanner) addToMem(kind TokenKind, pos int, tokenMem []*Token) []*Token {
	tokenMem = append(tokenMem, &Token{kind, l.fullStart, l.start, pos - l.fullStart, TokenCatNormal})
	l.fullStart = pos
	l.start = pos
	return tokenMem
}

func (l *LexerScanner) addToMemInPlace(kind TokenKind, pos int, length int, tokenMem []*Token) []*Token {
	tokenMem = append(tokenMem, &Token{kind, pos, pos, length, TokenCatNormal})
	return tokenMem
}

func (l *LexerScanner) createToken(kind TokenKind) *Token {
	return &Token{kind, l.fullStart, l.start, l.pos - l.fullStart, TokenCatNormal}
}

func (l *LexerScanner) scan(tokenMem []*Token) (*Token, []*Token) {
	l.fullStart = l.pos

	for {
		l.start = l.pos
		// handling end of file
		if l.pos >= l.eofPos {
			var current *Token
			if l.state != LexStateHtmlSection {
				current = l.createToken(EndOfFileToken)
			} else {
				current = &Token{InlineHtml, l.fullStart, l.fullStart, l.pos - l.fullStart, TokenCatNormal}
			}
			l.state = LexStateScriptSection
			if current.Kind == InlineHtml && l.pos-l.fullStart == 0 {
				continue
			}
			return current, tokenMem
		}

		if l.state == LexStateHtmlSection {
			// Keep scanning until we hit a script section Start tag
			if !isScriptStartTag(l.content, l.pos, l.eofPos) {
				l.pos++
				continue
			}
			l.state = LexStateScriptSection

			if l.pos-l.fullStart == 0 {
				continue
			}
			return &Token{InlineHtml, l.fullStart, l.fullStart, l.pos - l.fullStart, TokenCatNormal}, tokenMem
		}

		charCode := l.content[l.pos]

		if l.hereDocStatus == HereDocNowDoc {
			hasEncapsed := false
			for l.pos < l.eofPos {
				if l.pos+1 < l.eofPos && isNewLineChar(l.content[l.pos]) && isNowdocEnd(l.hereDocIdentifier, l.content, l.pos+1, l.eofPos) {
					l.pos += len(l.hereDocIdentifier) + 1
					if hasEncapsed {
						tokenMem = append(tokenMem, l.createToken(EncapsedAndWhitespace))
						l.start, l.fullStart = l.pos, l.pos
					}
					tokenMem = append(tokenMem, l.createToken(HeredocEnd))
					return l.createToken(-1), tokenMem
				} else {
					hasEncapsed = true
					l.pos++
					continue
				}
			}
			if hasEncapsed {
				tokenMem = append(tokenMem, l.createToken(EncapsedAndWhitespace))
				l.start, l.fullStart = l.pos, l.pos
			}

			return l.createToken(-1), tokenMem
		}

		switch charCode {

		case '#':
			scanSingleLineComment(l.content, &l.pos, l.eofPos, l.state)
			continue

		case ' ', '\t', '\r', '\n':
			l.pos++
			continue

		case '<', // <=>, <=, <<=, <<, < // TODO heredoc and nowdoc
			'.', // ..., .=, . // TODO also applies to floating point literals
			'=', // ===, ==, =
			'>', // >>=, >>, >=, >
			'*', // **=, **, *=, *
			'!', // !==, !=, !

			// Potential 2-char compound
			'+', // +=, ++, +
			'-', // -= , --, ->, -
			'%', // %=, %
			'^', // ^=, ^
			'|', // |=, ||, |
			'&', // &=, &&, &
			'?', // ??, ?, end-tag

			':', // : (TODO should this actually be treated as compound?)
			',', // , (TODO should this actually be treated as compound?)

			// Non-compound
			'@', '[', ']', '(',
			')', '{', '}', ';',
			'~', '\\':

			if isNowdocStart(l.content, l.pos, l.eofPos) || isHeredocStart(l.content, l.pos, l.eofPos) {
				tokenKind, ok := tryScanHeredocStart(l)
				if ok {
					return l.createToken(tokenKind), tokenMem
				}
			}

			if l.pos+1 < l.eofPos && charCode == '.' && isDigitChar(l.content[l.pos+1]) {
				kind := scanNumericLiteral(l.content, &l.pos, l.eofPos)
				return l.createToken(kind), tokenMem
			}

			// we must check for cast tokens
			if charCode == '(' {
				tokenKind, ok := tryScanCastToken(l)
				if ok {
					return l.createToken(tokenKind), tokenMem
				}
			}
			return scanOperatorOrPunctuactorToken(l), tokenMem

		case '/':
			if isSingleLineCommentStart(l.content, l.pos, l.eofPos) {
				scanSingleLineComment(l.content, &l.pos, l.eofPos, l.state)
				continue
			} else if isDelimitedCommentStart(l.content, l.pos, l.eofPos) {
				scanDelimitedComment(l.content, &l.pos, l.eofPos)
				continue
			} else if l.pos+1 < l.eofPos && l.content[l.pos+1] == '=' {
				l.pos += 2
				return l.createToken(SlashEqualsToken), tokenMem
			}
			l.pos++
			return l.createToken(SlashToken), tokenMem
		case '$':
			l.pos++
			if isNameStart(l.content, l.pos, l.eofPos) {
				scanName(l.content, &l.pos, l.eofPos)
				return l.createToken(VariableName), tokenMem
			}
			return l.createToken(DollarToken), tokenMem
		case '"', '\'':
			return getStringQuoteTokens(l, tokenMem)
		case 'b', 'B':
			if l.pos+1 < l.eofPos && (l.content[l.pos+1] == '\'' || l.content[l.pos+1] == '"') {
				l.pos++
				return getStringQuoteTokens(l, tokenMem)
			}
			return getNameOrDigitTokens(l, tokenMem)
		default:
			return getNameOrDigitTokens(l, tokenMem)
		}
	}
}
func isNowdocEnd(identifier string, content []rune, pos int, eof int) bool {
	l := len(identifier)
	if l+pos > eof {
		return false
	}
	runeIdentifier := []rune(identifier)
	for i := 0; i < l; i++ {
		if runeIdentifier[i] != content[pos+i] {
			return false
		}
	}
	return true
}

func isNowdocStart(content []rune, pos int, eof int) bool {
	// <<<'x'
	if pos+6 > eof {
		return false
	}
	return string(content[pos:pos+4]) == "<<<'"
}

func isHeredocStart(content []rune, pos int, eof int) bool {
	// <<<x
	if pos+5 > eof {
		return false
	}
	return string(content[pos:pos+3]) == "<<<"
}

func tryScanHeredocStart(l *LexerScanner) (TokenKind, bool) {
	foundTokenKind := Unknown

	pos := l.pos + 3 // consume <<<
	isNowDoc := l.content[pos] == '\''

	if isNowDoc {
		pos++
	}

	if isNameStart(l.content, pos, l.eofPos) == false {
		return foundTokenKind, false
	}
	pos++

	for ; pos < l.eofPos; pos++ {
		if isValidNameUnicodeChar(l.content[pos]) {
			continue
		} else if l.content[pos] == '\'' && isNowDoc == false {
			return foundTokenKind, false
		} else if l.content[pos] == '\'' && isNowDoc == true {
			if pos+1 < l.eofPos && isNewLineChar(l.content[pos+1]) {
				l.hereDocIdentifier = string(l.content[l.pos+4 : pos+1])
				l.pos = pos + 1
				l.hereDocStatus = HereDocNowDoc
				return HeredocStart, true
			}
		} else if isNewLineChar(l.content[pos]) {
			l.hereDocIdentifier = string(l.content[l.pos+3 : pos+1])
			l.pos = pos
			l.hereDocStatus = HereDocNormal
			return HeredocStart, true
		}
	}
	return foundTokenKind, false
}

func tryScanCastToken(l *LexerScanner) (TokenKind, bool) {
	foundTokenKind := Unknown
	for i := l.pos + 1; i < l.eofPos; i++ {
		if unicode.IsSpace(l.content[i]) {
			continue
		}

		if foundTokenKind != Unknown && l.content[i] == ')' {
			l.pos = i + 1
			return foundTokenKind, true
		}

		if foundTokenKind != Unknown && l.content[i] != ')' {
			return foundTokenKind, false
		}

		// no name start return false
		if !isNameStart(l.content, i, l.eofPos) {
			return foundTokenKind, false
		}

		// lookahead for cast keywords
		for _, castString := range CAST_KEYWORDS {
			if i+len(castString) >= l.eofPos {
				continue
			}

			word := strings.ToLower(string(l.content[i : i+len(castString)]))
			if word == castString {
				foundTokenKind = CAST_KEYWORDS_MAP[castString]
				i = i + len(castString) - 1
				break
			}
		}
	}
	return foundTokenKind, false
}

func tryScanYieldFrom(l *LexerScanner) (int, bool) {
	foundTokenKind := false
	from := "from"
	fromLen := len(from)
	for i := l.pos + 1; i < l.eofPos; i++ {

		if unicode.IsSpace(l.content[i]) || l.content[i] == ';' {
			if foundTokenKind {
				return i, true
			}
			continue
		}

		if i+fromLen >= l.eofPos {
			return -1, false
		}

		// no name start return false
		if !isNameStart(l.content, i, l.eofPos) {
			return -1, false
		}

		word := strings.ToLower(string(l.content[i : i+fromLen]))
		if word == from {
			foundTokenKind = true
			i = i + fromLen - 1
		}
	}
	return -1, false
}

func getNameOrDigitTokens(l *LexerScanner, tokenMem []*Token) (*Token, []*Token) {
	if isNameStart(l.content, l.pos, l.eofPos) {
		scanName(l.content, &l.pos, l.eofPos)
		token := l.createToken(Name)
		tokenText := token.getText(l.content)
		lowerText := strings.ToLower(tokenText)
		if isKeywordOrReservedWordStart(lowerText) {
			token = getKeywordOrReservedWordTokenFromNameToken(token, lowerText, l.content, &l.pos, l.eofPos)
			if token.Kind == YieldKeyword {
				newPos, ok := tryScanYieldFrom(l)
				if ok {
					l.pos = newPos
					token = l.createToken(YieldFromKeyword)
				}
			}
		}
		return token, tokenMem
	} else if isDigitChar(l.content[l.pos]) {
		kind := scanNumericLiteral(l.content, &l.pos, l.eofPos)
		return l.createToken(kind), tokenMem
	}
	l.pos++
	return l.createToken(Unknown), tokenMem
}

func getStringQuoteTokens(l *LexerScanner, tokenMem []*Token) (*Token, []*Token) {
	if l.content[l.pos] == '"' {
		tokenMem = scanTemplateAndSetTokenValue(l, tokenMem)
		return l.createToken(-1), tokenMem
	}
	l.pos++
	if scanStringLiteral(l.content, &l.pos, l.eofPos) {
		return l.createToken(StringLiteralToken), tokenMem
	}
	return l.createToken(EncapsedAndWhitespace), tokenMem
}

func isScriptStartTag(text []rune, pos int, eofPos int) bool {

	if text[pos] != '<' {
		return false
	}

	if pos+3 > eofPos {
		return false
	}

	if pos+5 < eofPos {
		start := strings.ToLower(string(text[pos : 5+pos]))
		end := text[pos+5]
		if start == "<?php" {
			switch end {
			case '\n',
				'\r',
				' ',
				'\t':
				return true
			}
		}
	}

	if string(text[pos:pos+3]) == "<?=" {
		return true
	}
	return false
}

func scanOperatorOrPunctuactorToken(lexer *LexerScanner) *Token {
	// TODO this can be made more performant, but we're going for simple/correct first.
	// TODO
	for tokenEnd := 6; tokenEnd >= 0; tokenEnd-- {
		if lexer.pos+tokenEnd >= lexer.eofPos {
			continue
		}
		// TODO get rid of strtolower for perf reasons
		textSubstring := strings.ToLower(string(lexer.content[lexer.pos : lexer.pos+tokenEnd+1]))
		if tokenKind, ok := OPERATORS_AND_PUNCTUATORS[textSubstring]; ok {
			if tokenKind == ScriptSectionStartTag {
				if lexer.state == LexStateScriptSectionParsed {
					continue
				}
				lexer.state = LexStateScriptSectionParsed
			}
			lexer.pos += tokenEnd + 1
			if tokenKind == ScriptSectionEndTag {
				lexer.state = LexStateHtmlSection
			}
			return lexer.createToken(tokenKind)
		}
	}
	panic("Unknown token Kind in OPERATORS_AND_PUNCTUATORS")
}

func getKeywordOrReservedWordTokenFromNameToken(token *Token, lowerKeywordStart string, text []rune, pos *int, eofPos int) *Token {

	kind, ok := KEYWORDS[lowerKeywordStart]
	if !ok {
		kind, ok = RESERVED_WORDS[lowerKeywordStart]
	}
	token.Kind = kind
	return token
}

func isDigitChar(at rune) bool {
	return at >= '0' &&
		at <= '9'
}

func isKeywordOrReservedWordStart(text string) bool {
	_, ok := KEYWORDS[text]
	_, ok2 := RESERVED_WORDS[text]
	return ok || ok2
}

func scanStringLiteral(text []rune, pos *int, eofPos int) bool {
	isTerminated := false
	for *pos < eofPos {
		if isSingleQuoteEscapeSequence(text, *pos) {
			*pos += 2
			continue
		} else if text[*pos] == '\'' {
			*pos++
			isTerminated = true
			break
		} else {
			*pos++
			continue
		}
	}

	return isTerminated
}

func scanDelimitedComment(text []rune, pos *int, eofPos int) {
	for *pos < eofPos {
		if *pos+1 < eofPos && text[*pos] == '*' && text[*pos+1] == '/' {
			*pos += 2
			return
		}
		*pos++
	}

}

func scanName(text []rune, pos *int, eofPos int) {
	for *pos < eofPos {
		charCode := text[*pos]
		if isNameNonDigitChar(charCode) || isDigitChar(charCode) {
			*pos++
			continue
		}
		return
	}
}

func scanTemplateAndSetTokenValue(l *LexerScanner, tokenMem []*Token) []*Token {
	startPosition := l.start
	eofPos := l.eofPos
	pos := &l.pos
	fileContent := l.content
	*pos++
	for {
		if *pos >= eofPos {
			// UNTERMINATED, report error
			if len(tokenMem) == 0 {
				tokenMem = append(tokenMem, &Token{DoubleQuoteToken, l.fullStart, l.start, l.start - l.fullStart + 1, TokenCatNormal})
				l.fullStart = l.start
				if l.start != eofPos-1 {
					tokenMem = append(tokenMem, &Token{EncapsedAndWhitespace, l.fullStart, l.start + 1, *pos - l.fullStart, TokenCatNormal})
				}

				return tokenMem
			} else {
				return tokenMem
			}
		}

		char := l.content[*pos]

		if char == '"' {

			if len(tokenMem) == 0 {
				*pos++
				tokenMem = l.addToMem(StringLiteralToken, *pos, tokenMem)
				return tokenMem
				//return NoSubstitutionTemplateLiteral
			} else {
				if *pos-l.fullStart > 0 {
					tokenMem = l.addToMem(EncapsedAndWhitespace, *pos, tokenMem)
				}
				*pos++
				tokenMem = l.addToMem(DoubleQuoteToken, *pos, tokenMem)
				return tokenMem
			}
		}

		if char == '$' {
			if isNameStart(fileContent, *pos+1, eofPos) {
				if len(tokenMem) == 0 {
					tokenMem = append(tokenMem, &Token{DoubleQuoteToken, l.fullStart, startPosition, startPosition - l.fullStart + 1, TokenCatNormal})
					l.start++
					l.fullStart = l.start
				}
				if *pos-l.start > 0 {
					tokenMem = l.addToMem(EncapsedAndWhitespace, *pos, tokenMem)
				}
				*pos++
				scanName(fileContent, pos, eofPos)
				tokenMem = l.addToMem(VariableName, *pos, tokenMem)

				if *pos < eofPos && fileContent[*pos] == '[' {
					*pos++
					tokenMem = l.addToMem(OpenBracketToken, *pos, tokenMem)
					if isDigitChar(fileContent[*pos]) {
						*pos++
						scanName(fileContent, pos, eofPos)
						tokenMem = l.addToMem(IntegerLiteralToken, *pos, tokenMem)
					} else if isNameStart(fileContent, *pos, eofPos) {
						// var name index
						*pos++
						scanName(fileContent, pos, eofPos)
						tokenMem = l.addToMem(Name, *pos, tokenMem)
					}
					if fileContent[*pos] == ']' {
						*pos++
						tokenMem = l.addToMem(CloseBracketToken, *pos, tokenMem)
					}
				} else if *pos+1 < eofPos && fileContent[*pos] == '-' && fileContent[*pos+1] == '>' {
					if isNameStart(fileContent, *pos+2, eofPos) {
						*pos++
						*pos++
						tokenMem = l.addToMem(ArrowToken, *pos, tokenMem)
						// var name index
						*pos++
						scanName(fileContent, pos, eofPos)
						tokenMem = l.addToMem(Name, *pos, tokenMem)
					}
				}

				continue
			} else if *pos+1 < eofPos && fileContent[*pos+1] == '{' {
				// curly
				var exit bool
				if exit, tokenMem = saveCurlyExpression(l, DollarOpenBraceToken, pos, startPosition, tokenMem); exit {
					return tokenMem
				}
				continue
			}
		}

		if char == '{' {
			if *pos+1 < eofPos && fileContent[*pos+1] == '$' {
				var exit bool
				if exit, tokenMem = saveCurlyExpression(l, OpenBraceDollarToken, pos, startPosition, tokenMem); exit {
					return tokenMem
				}
				continue
			}
		}

		// Escape character
		if char == '\\' {
			*pos++
			scanDqEscapeSequence(fileContent, pos, eofPos)
			continue
		}

		*pos++
	}
	return tokenMem
}

func saveCurlyExpression(l *LexerScanner, openToken TokenKind, pos *int, startPosition int, tokenMem []*Token) (bool, []*Token) {
	if len(tokenMem) == 0 {
		tokenMem = append(tokenMem, &Token{DoubleQuoteToken, l.fullStart, startPosition, startPosition - l.fullStart + 1, TokenCatNormal})
		l.start++
		l.fullStart = l.start
	}
	if *pos-l.start > 0 {
		tokenMem = l.addToMem(EncapsedAndWhitespace, *pos, tokenMem)
	}
	openTokenLen := 1
	if openToken == DollarOpenBraceToken {
		openTokenLen = 2
	}
	tokenMem = l.addToMemInPlace(openToken, *pos, openTokenLen, tokenMem)
	*pos += openTokenLen
	l.fullStart = *pos
	l.start = *pos
	isFirst := true
	for *pos < l.eofPos {
		t, tokenMemTmp := l.scan(nil)
		l.fullStart = *pos
		l.start = *pos

		if t.Kind == -1 {
			tokenMem = append(tokenMem, tokenMemTmp...)
			continue
		}

		if isFirst && (t.Kind == Name || IsKeywordOrReserverdWordToken(t.Kind)) {
			t.Kind = StringVarname
			isFirst = false
		}
		if t.Kind == VariableName {
			isFirst = false
		}

		tokenMem = append(tokenMem, t)
		if t.Kind == ScriptSectionEndTag {
			return true, tokenMem
		}
		if t.Kind == CloseBraceToken {
			break
		}

	}
	return false, tokenMem
}

func scanDqEscapeSequence(text []rune, pos *int, eofPos int) {
	if *pos >= eofPos {
		return
	}
	char := text[*pos]
	switch char {
	// dq-simple-escape-sequence
	case '"',
		'\\',
		'$',
		'e',
		'f',
		'r',
		't',
		'v':
		*pos++
		return

		// dq-hexadecimal-escape-sequence
	case 'x',
		'X':
		*pos++
		for i := 0; i < 2; i++ {
			if isHexadecimalDigit(text[*pos]) {
				*pos++
			}
		}
		return

		// dq-unicode-escape-sequence
	case 'u':
		*pos++
		if text[*pos] == '{' {
			scanHexadecimalLiteral(text, pos, eofPos)
			if text[*pos] == '}' {
				*pos++
				return
			}
			// OTHERWISE ERROR
		}
		return
	default:
		// dq-octal-digit-escape-sequence
		if isOctalDigitChar(text[*pos]) {
			for i := *pos; i < *pos+3; i++ {
				if isOctalDigitChar(text[*pos]) {
					return
				}
				*pos++
				return
			}
		}

		*pos++
		return
	}
}

func scanOctalLiteral(text []rune, pos *int, eofPos int) bool {
	isValid := true
	for *pos < eofPos {
		charCode := text[*pos]
		if isOctalDigitChar(charCode) {
			*pos++
			continue
		} else if isDigitChar(charCode) {
			*pos++
			isValid = false
			continue
		}
		break
	}
	return isValid
}

func scanDecimalLiteral(text []rune, pos *int, eofPos int) {
	for *pos < eofPos {
		charCode := text[*pos]
		if isDigitChar(charCode) {
			*pos++
			continue
		}
		return
	}
}
func scanSingleLineComment(text []rune, pos *int, eofPos int, state LexerState) {
	for *pos < eofPos {
		if isNewLineChar(text[*pos]) || isScriptEndTag(text, *pos, state) {
			return
		}
		*pos++
	}
}
func isSingleLineCommentStart(text []rune, pos int, eofPos int) bool {
	return pos+1 < eofPos && text[pos] == '/' && text[pos+1] == '/'
}

func isSingleQuoteEscapeSequence(text []rune, pos int) bool {
	return text[pos] == '\\' &&
		('\'' == text[pos+1] || '\\' == text[pos+1])
}

func isScriptEndTag(text []rune, pos int, state LexerState) bool {
	if state != LexStateScriptSection && text[pos] == '?' && text[pos+1] == '>' {
		return true
	}
	return false
}

func isNewLineChar(charCode rune) bool {
	return charCode == '\n' || charCode == '\r'
}

func isOctalDigitChar(charCode rune) bool {
	return charCode >= '0' &&
		charCode <= '7'
}

func isBinaryDigitChar(charCode rune) bool {
	return charCode == '0' ||
		charCode == '1'
}

func isHexadecimalDigit(charCode rune) bool {
	// 0  1  2  3  4  5  6  7  8  9
	// a  b  c  d  e  f
	// A  B  C  D  E  F
	return charCode >= '0' && charCode <= '9' || charCode >= 'a' && charCode <= 'f' || charCode >= 'A' && charCode <= 'F'
}

func isNameNonDigitChar(charCode rune) bool {
	return isNonDigitChar(charCode) || isValidNameUnicodeChar(charCode)
}

func isNonDigitChar(charCode rune) bool {
	return (charCode >= 'a' && charCode <= 'z') ||
		(charCode >= 'A' && charCode <= 'Z') ||
		charCode == '_'
}

func isValidNameUnicodeChar(charCode rune) bool {
	return unicode.IsLetter(charCode)
}

func scanHexadecimalLiteral(text []rune, pos *int, eofPos int) bool {
	isValid := true
	p := *pos
	for p < eofPos {
		charCode := text[*pos]
		if isHexadecimalDigit(charCode) {
			p++
			*pos++
			continue
		} else if isDigitChar(charCode) || isNameNonDigitChar(charCode) {
			// REPORT ERROR;
			p++
			isValid = false
			continue
		}
		break
	}
	return isValid
}

func scanFloatingPointLiteral(text []rune, pos *int, eofPos int) bool {
	hasDot := false
	var expStart int = -1
	hasSign := false
	for *pos < eofPos {
		char := text[*pos]
		if isDigitChar(char) {
			*pos++
			continue
		} else if char == '.' {
			if hasDot || expStart != -1 {
				// Dot not valid, done scanning
				break
			}
			hasDot = true
			*pos++
			continue
		} else if char == 'e' || char == 'E' {
			if expStart != -1 {
				// exponential not valid here, done scanning
				break
			}
			expStart = *pos
			*pos++
			continue
		} else if char == '+' || char == '-' {
			if expStart != -1 && expStart == (*pos)-1 {
				hasSign = true
				*pos++
				continue
			}
			// sign not valid here, done scanning
			break
		}
		// unexpected character, done scanning
		break
	}

	if expStart != -1 {
		expectedMinPos := expStart + 2
		if hasSign {
			expectedMinPos = expStart + 3
		}

		if *pos >= expectedMinPos {
			return true
		}
		// exponential is invalid, reset position
		*pos = expStart
	}

	return hasDot
}

func scanBinaryLiteral(text []rune, pos *int, eofPos int) bool {
	isValid := true
	for *pos < eofPos {
		charCode := text[*pos]
		if isBinaryDigitChar(charCode) {
			*pos++
			continue
		} else if isDigitChar(charCode) {
			*pos++
			// REPORT ERROR;
			isValid = false
			continue
		}
		break
	}
	return isValid
}

func isNameStart(text []rune, pos int, eofPos int) bool {
	return pos < eofPos && isNameNonDigitChar(text[pos])
}

func isDelimitedCommentStart(text []rune, pos int, eofPos int) bool {
	return pos+1 < eofPos && text[pos] == '/' && text[pos+1] == '*'
}

func isHexadecimalLiteralStart(text []rune, pos int, eofPos int) bool {
	// 0x  0X
	return pos+1 < eofPos &&
		text[pos] == '0' &&
		(text[pos+1] == 'x' || text[pos+1] == 'X')
}

func isBinaryLiteralStart(text []rune, pos int, eofPos int) bool {
	return pos+1 < eofPos && text[pos] == '0' && (text[pos+1] == 'b' || text[pos+1] == 'B')
}

func scanNumericLiteral(text []rune, pos *int, eofPos int) TokenKind {
	var prevPos int

	if isBinaryLiteralStart(text, *pos, eofPos) {
		*pos += 2
		prevPos = *pos
		isValidBinaryLiteral := scanBinaryLiteral(text, pos, eofPos)
		if prevPos == *pos || !isValidBinaryLiteral {
			// invalid binary literal
			return InvalidBinaryLiteral
		}
		return IntegerLiteralToken
		//return BinaryLiteralToken
	} else if isHexadecimalLiteralStart(text, *pos, eofPos) {
		*pos += 2

		isValidHexLiteral := scanHexadecimalLiteral(text, pos, eofPos)
		if !isValidHexLiteral {
			return IntegerLiteralToken
			// invalid hexadecimal literal
		}
		return IntegerLiteralToken
		//return HexadecimalLiteralToken
	} else if isDigitChar(text[*pos]) || text[*pos] == '.' {
		// TODO throw error if there is no number past the dot.
		prevPos = *pos
		isValidFloatingLiteral := scanFloatingPointLiteral(text, pos, eofPos)
		if isValidFloatingLiteral {
			return FloatingLiteralToken
		}

		// Reset, try scanning octal literal
		*pos = prevPos
		if text[*pos] == '0' {
			isValidOctalLiteral := scanOctalLiteral(text, pos, eofPos)

			// Check that it's not a 0 decimal literal
			if *pos == prevPos+1 {
				return IntegerLiteralToken
			}

			if !isValidOctalLiteral {
				return InvalidOctalLiteralToken
			}
			return IntegerLiteralToken
			//return OctalLiteralToken
		}

		scanDecimalLiteral(text, pos, eofPos)
		return IntegerLiteralToken
	}

	return Unknown
}
