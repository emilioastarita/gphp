package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"
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
	content           []byte
	stringDelimiter   TokenKind
}

type TokensStream struct {
	Tokens []*Token
	Pos    int
	EofPos int
	tokenMem []*Token
	lexer    LexerScanner
}

func (s *TokensStream) Source(content []byte) {
	s.lexer = LexerScanner{
		LexStateHtmlSection,
		HereDocStateNone,
		"",
		0,
		0,
		0,
		0,
		content,
		DoubleQuoteToken,
	}
	s.lexer.eofPos = len(s.lexer.content)
}

func (s *TokensStream) CreateTokens() {
	lexer := s.lexer
	token := &Token{}
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

func (s *TokensStream) Serialize() []TokenCompareForm {
	tokens := make([]TokenCompareForm, 0)
	for _, token := range s.Tokens {
		b := token.getFullFormCompare()
		tokens = append(tokens, b)
	}
	return tokens
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
			return l.createToken(-1), parseDocNow(l, tokenMem)
		} else if l.hereDocStatus == HereDocNormal {
			// @todo We are handling heredoc as docnow for now
			return l.createToken(-1), parseHeredoc(l, tokenMem)
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

			if l.pos+1 < l.eofPos && charCode == '.' && isDigitChar(rune(l.content[l.pos+1])) {
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
		case '"', '\'', '`':
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

func parseDocNow(l *LexerScanner, tokenMem []*Token) []*Token {
	hasEncapsed := false
	l.hereDocStatus = HereDocStateNone
	for l.pos < l.eofPos {
		if l.pos+1 < l.eofPos && isNewLineChar(rune(l.content[l.pos])) && isNowdocEnd(l.hereDocIdentifier, l.content, l.pos+1, l.eofPos) {
			if hasEncapsed {
				l.pos++
				tokenMem = append(tokenMem, l.createToken(EncapsedAndWhitespace))
				l.start, l.fullStart = l.pos, l.pos
			}
			l.pos += len(l.hereDocIdentifier)
			tokenMem = append(tokenMem, l.createToken(HeredocEnd))
			l.start, l.fullStart = l.pos, l.pos
			return tokenMem
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

	return tokenMem
}

func parseHeredoc(l *LexerScanner, tokenMem []*Token) []*Token {
	l.hereDocStatus = HereDocStateNone
	startPosition := l.start
	eofPos := l.eofPos
	pos := &l.pos
	fileContent := l.content
	for {
		if *pos >= eofPos {
			// UNTERMINATED, report error
			tokenMem = append(tokenMem, &Token{EncapsedAndWhitespace, l.fullStart, l.start, *pos - l.fullStart, TokenCatNormal})
			return tokenMem
		}

		char := l.content[*pos]

		if *pos+1 < l.eofPos && isNewLineChar(rune(l.content[*pos])) && isNowdocEnd(l.hereDocIdentifier, l.content, *pos+1, l.eofPos) {
			tokenMem = append(tokenMem, &Token{EncapsedAndWhitespace, l.fullStart, l.start, *pos - l.fullStart + 1, TokenCatNormal})
			*pos++
			l.start, l.fullStart = *pos, *pos
			*pos += len(l.hereDocIdentifier)
			tokenMem = l.addToMem(HeredocEnd, *pos, tokenMem)
			l.start, l.fullStart = *pos, *pos
			return tokenMem
		}

		if char == '$' {
			if isNameStart(fileContent, *pos+1, eofPos) {
				if *pos-l.start > 0 {
					tokenMem = l.addToMem(EncapsedAndWhitespace, *pos, tokenMem)
				}
				*pos++
				scanName(fileContent, pos, eofPos)
				tokenMem = l.addToMem(VariableName, *pos, tokenMem)

				if *pos < eofPos && fileContent[*pos] == '[' {
					*pos++
					tokenMem = l.addToMem(OpenBracketToken, *pos, tokenMem)
					if isDigitChar(rune(fileContent[*pos])) {
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
				if exit, tokenMem = saveCurlyExpressionHereDoc(l, DollarOpenBraceToken, pos, startPosition, tokenMem); exit {
					return tokenMem
				}
				continue
			}
		}

		if char == '{' {
			if *pos+1 < eofPos && fileContent[*pos+1] == '$' {
				var exit bool
				if exit, tokenMem = saveCurlyExpressionHereDoc(l, OpenBraceDollarToken, pos, startPosition, tokenMem); exit {
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

func isNowdocEnd(identifier string, content []byte, pos int, eof int) bool {
	l := len(identifier)
	if l+pos > eof {
		return false
	}
	runeIdentifier := []byte(identifier)
	for i := 0; i < l; i++ {
		if runeIdentifier[i] != content[pos+i] {
			return false
		}
	}
	return true
}

func isNowdocStart(content []byte, pos int, eof int) bool {
	// <<<'x'
	if pos+6 > eof {
		return false
	}
	return string(content[pos:pos+4]) == "<<<'"
}

func isHeredocStart(content []byte, pos int, eof int) bool {
	// <<<x
	if pos+5 > eof {
		return false
	}
	return string(content[pos:pos+3]) == "<<<"
}

func tryScanHeredocStart(l *LexerScanner) (TokenKind, bool) {
	foundTokenKind := Unknown

	pos := l.pos + 3 // consume <<<

	for unicode.IsSpace(rune(l.content[pos])) && !isNewLineChar(rune(l.content[pos])) {
		pos++
	}

	isNowDoc := l.content[pos] == '\''

	if isNowDoc {
		pos++
	}

	if isNameStart(l.content, pos, l.eofPos) == false {
		return foundTokenKind, false
	}
	startIdentifier := pos
	pos++

	for ; pos < l.eofPos; {

		charCode, size := utf8.DecodeRune(l.content[pos:])

		if isValidNameUnicodeChar(charCode) {
			pos += size
			continue
		} else if l.content[pos] == '\'' && isNowDoc == false {
			return foundTokenKind, false
		} else if l.content[pos] == '\'' && isNowDoc == true {
			if pos+1 < l.eofPos && isNewLineChar(rune(l.content[pos+1])) {
				l.hereDocIdentifier = string(l.content[startIdentifier:pos])
				l.pos = pos + 2
				l.hereDocStatus = HereDocNowDoc
				return HeredocStart, true
			}
		} else if isNewLineChar(rune(l.content[pos])) {
			l.hereDocIdentifier = string(l.content[startIdentifier:pos])
			l.pos = pos + 1
			l.hereDocStatus = HereDocNormal
			return HeredocStart, true
		}
	}
	return foundTokenKind, false
}

func tryScanCastToken(l *LexerScanner) (TokenKind, bool) {
	foundTokenKind := Unknown
	for i := l.pos + 1; i < l.eofPos; i++ {
		if unicode.IsSpace(rune(l.content[i])) {
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
		if foundTokenKind != Unknown {
			continue
		}
		return foundTokenKind, false

	}
	return foundTokenKind, false
}

func tryScanYieldFrom(l *LexerScanner) (int, bool) {
	foundTokenKind := false
	foundPos := -1
	from := "from"
	fromLen := len(from)

	for i := l.pos + 1; i < l.eofPos; i++ {

		if unicode.IsSpace(rune(l.content[i])) || l.content[i] == ';' {
			if foundTokenKind {
				return i, true
			}
			continue
		}

		if i+fromLen > l.eofPos {
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
			foundPos = i + 1
		}
	}
	return foundPos, foundTokenKind
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
	} else if isDigitChar(rune(l.content[l.pos])) {
		kind := scanNumericLiteral(l.content, &l.pos, l.eofPos)
		return l.createToken(kind), tokenMem
	}
	l.pos++
	return l.createToken(Unknown), tokenMem
}

func getStringQuoteTokens(l *LexerScanner, tokenMem []*Token) (*Token, []*Token) {
	if l.content[l.pos] == '"' || l.content[l.pos] == '`' {
		l.stringDelimiter = DoubleQuoteToken
		if l.content[l.pos] == '`' {
			l.stringDelimiter = BacktickToken
		}
		tokenMem = scanTemplateAndSetTokenValue(l, tokenMem)
		return l.createToken(-1), tokenMem
	}
	l.pos++
	if scanStringLiteral(l.content, &l.pos, l.eofPos) {
		return l.createToken(StringLiteralToken), tokenMem
	}
	return l.createToken(EncapsedAndWhitespace), tokenMem
}

func isScriptStartTag(text []byte, pos int, eofPos int) bool {

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

func getKeywordOrReservedWordTokenFromNameToken(token *Token, lowerKeywordStart string, text []byte, pos *int, eofPos int) *Token {

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

func scanStringLiteral(text []byte, pos *int, eofPos int) bool {
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

func scanDelimitedComment(text []byte, pos *int, eofPos int) {
	for *pos < eofPos {
		if *pos+1 < eofPos && text[*pos] == '*' && text[*pos+1] == '/' {
			*pos += 2
			return
		}
		*pos++
	}

}

func scanName(text []byte, pos *int, eofPos int) {
	for *pos < eofPos {
		charCode, size := utf8.DecodeRune(text[*pos:])
		if isNameNonDigitChar(charCode) || isDigitChar(charCode) {
			*pos += size
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
				tokenMem = append(tokenMem, &Token{l.stringDelimiter, l.fullStart, l.start, l.start - l.fullStart + 1, TokenCatNormal})
				l.start++
				l.fullStart = l.start
				if l.start != eofPos {
					tokenMem = append(tokenMem, &Token{EncapsedAndWhitespace, l.fullStart, l.start, *pos - l.fullStart, TokenCatNormal})
				}

				return tokenMem
			} else {
				return tokenMem
			}
		}

		char := l.content[*pos]

		if char == '"' && l.stringDelimiter == DoubleQuoteToken || char == '`' && l.stringDelimiter == BacktickToken {
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
				tokenMem = l.addToMem(l.stringDelimiter, *pos, tokenMem)
				return tokenMem
			}
		}

		if char == '$' {
			if isNameStart(fileContent, *pos+1, eofPos) {
				if len(tokenMem) == 0 {
					tokenMem = append(tokenMem, &Token{l.stringDelimiter, l.fullStart, startPosition, startPosition - l.fullStart + 1, TokenCatNormal})
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
					if isDigitChar(rune(fileContent[*pos])) {
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
		tokenMem = append(tokenMem, &Token{l.stringDelimiter, l.fullStart, startPosition, startPosition - l.fullStart + 1, TokenCatNormal})
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

func saveCurlyExpressionHereDoc(l *LexerScanner, openToken TokenKind, pos *int, startPosition int, tokenMem []*Token) (bool, []*Token) {
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

func scanDqEscapeSequence(text []byte, pos *int, eofPos int) {
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
			if isHexadecimalDigit(rune(text[*pos])) {
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
		if isOctalDigitChar(rune(text[*pos])) {
			for i := *pos; i < *pos+3; i++ {
				if isOctalDigitChar(rune(text[*pos])) {
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

func scanOctalLiteral(text []byte, pos *int, eofPos int) bool {
	isValid := true
	for *pos < eofPos {
		charCode := text[*pos]
		if isOctalDigitChar(rune(charCode)) {
			*pos++
			continue
		} else if isDigitChar(rune(charCode)) {
			*pos++
			isValid = false
			continue
		}
		break
	}
	return isValid
}

func scanDecimalLiteral(text []byte, pos *int, eofPos int) {
	for *pos < eofPos {
		charCode := text[*pos]
		if isDigitChar(rune(charCode)) {
			*pos++
			continue
		}
		return
	}
}
func scanSingleLineComment(text []byte, pos *int, eofPos int, state LexerState) {
	for *pos < eofPos {
		if isNewLineChar(rune(text[*pos])) || isScriptEndTag(text, *pos, state) {
			return
		}
		*pos++
	}
}
func isSingleLineCommentStart(text []byte, pos int, eofPos int) bool {
	return pos+1 < eofPos && text[pos] == '/' && text[pos+1] == '/'
}

func isSingleQuoteEscapeSequence(text []byte, pos int) bool {
	return text[pos] == '\\' &&
		('\'' == text[pos+1] || '\\' == text[pos+1])
}

func isScriptEndTag(text []byte, pos int, state LexerState) bool {
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
	switch charCode {
	case '/', '<', '.', '=',
		'>', '*', '!', '-', '$',
		'"', '%', '^', '|', '&', '?',
		':', ',', '@', '{', '}', ';',
		'~', '\\', '[', ']', '\'',
		'+', ')', '(':
		return false
	}

	return (charCode >= '\u0080' || charCode <= '\u00ff') &&
		!unicode.IsSpace(charCode) &&
		!unicode.IsDigit(charCode)

	//return (charCode >= 'a' && charCode <= 'z') ||
	//	(charCode >= 'A' && charCode <= 'Z') ||
	//	charCode == '_'
}

func isValidNameUnicodeChar(charCode rune) bool {
	return unicode.IsLetter(charCode)
}

func scanHexadecimalLiteral(text []byte, pos *int, eofPos int) bool {
	isValid := true
	p := *pos
	for p < eofPos {
		charCode := rune(text[*pos])
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

func scanFloatingPointLiteral(text []byte, pos *int, eofPos int) bool {
	hasDot := false
	expStart := -1
	hasSign := false
	for *pos < eofPos {
		char := rune(text[*pos])
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

func scanBinaryLiteral(text []byte, pos *int, eofPos int) bool {

	for *pos < eofPos {
		charCode := rune(text[*pos])
		if isBinaryDigitChar(charCode) {
			*pos++
			continue
		} else if isDigitChar(charCode) {
			return false
		}
		break
	}
	return true
}

func isNameStart(text []byte, pos int, eofPos int) bool {
	charCode, _ := utf8.DecodeRune(text[pos:])
	return pos < eofPos && isNameNonDigitChar(charCode)
}

func isDelimitedCommentStart(text []byte, pos int, eofPos int) bool {
	return pos+1 < eofPos && text[pos] == '/' && text[pos+1] == '*'
}

func isHexadecimalLiteralStart(text []byte, pos int, eofPos int) bool {
	// 0x  0X
	return pos+1 < eofPos &&
		text[pos] == '0' &&
		(text[pos+1] == 'x' || text[pos+1] == 'X')
}

func isBinaryLiteralStart(text []byte, pos int, eofPos int) bool {
	return pos+1 < eofPos && text[pos] == '0' && (text[pos+1] == 'b' || text[pos+1] == 'B')
}

func scanNumericLiteral(text []byte, pos *int, eofPos int) TokenKind {
	var prevPos int

	if isBinaryLiteralStart(text, *pos, eofPos) {
		*pos += 2
		prevPos = *pos
		isValidBinaryLiteral := scanBinaryLiteral(text, pos, eofPos)
		if prevPos == *pos || !isValidBinaryLiteral {
			// invalid binary literal
			return IntegerLiteralToken
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
	} else if isDigitChar(rune(text[*pos])) || text[*pos] == '.' {
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
