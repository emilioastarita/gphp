package lexer

import (
	"encoding/json"
	"os"
	"strings"
	"unicode"
)

type LexerState int

const (
	LexStateHtmlSection LexerState = iota
	LexStateScriptSection
	LexStateScriptSectionParsed
)

type LexerScanner struct {
	state     LexerState
	pos       int
	eofPos    int
	fullStart int
	start     int
	content   []rune
}

type TokensStream struct {
	Tokens []Token
	Pos    int
	EofPos int

	tokenMem []Token
	lexer    LexerScanner
}

func (s *TokensStream) Source(content string) {
	s.lexer = LexerScanner{
		LexStateHtmlSection,
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
			s.Tokens = append(s.Tokens, *token)
			lexer.pos = token.fullStart + token.length
		}
	}
	s.Pos = 0
	s.EofPos = len(s.Tokens) - 1
}

func (s *TokensStream) ScanNext() Token {
	if s.Pos >= s.EofPos {
		return s.Tokens[s.EofPos]
	}
	pos := s.Pos
	s.Pos++
	return s.Tokens[pos]
}

func (s *TokensStream) Debug() {
	for _, token := range s.Tokens {
		b, _ := json.MarshalIndent(token.getShortForm([]rune(s.lexer.content)), "", "    ")
		os.Stdout.Write(b)
		println("")
	}
}

func (l *LexerScanner) addToMem(kind TokenKind, pos int, tokenMem []Token) []Token {
	tokenMem = append(tokenMem, Token{kind, l.fullStart, l.start, pos - l.fullStart})
	l.fullStart = pos
	l.start = pos
	return tokenMem
}

func (l *LexerScanner) addToMemInPlace(kind TokenKind, pos int, length int, tokenMem []Token) []Token {
	tokenMem = append(tokenMem, Token{kind, pos, pos, length})
	return tokenMem
}

func (l *LexerScanner) createToken(kind TokenKind) *Token {
	return &Token{kind, l.fullStart, l.start, l.pos - l.fullStart}
}

func (l *LexerScanner) scan(tokenMem []Token) (*Token, []Token) {
	l.fullStart = l.pos

	for {
		l.start = l.pos
		// handling end of file
		if l.pos >= l.eofPos {
			var current *Token
			if l.state != LexStateHtmlSection {
				current = l.createToken(EndOfFileToken)
			} else {
				current = &Token{InlineHtml, l.fullStart, l.fullStart, l.pos - l.fullStart}
			}
			l.state = LexStateScriptSection
			if current.Kind == InlineHtml && l.pos-l.fullStart == 0 {
				continue
			}
			return current, tokenMem
		}

		if l.state == LexStateHtmlSection {
			// Keep scanning until we hit a script section start tag
			if !isScriptStartTag(l.content, l.pos, l.eofPos) {
				l.pos++
				continue
			}
			l.state = LexStateScriptSection

			if l.pos-l.fullStart == 0 {
				continue
			}
			return &Token{InlineHtml, l.fullStart, l.fullStart, l.pos - l.fullStart}, tokenMem
		}

		charCode := l.content[l.pos]

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

			if charCode == '.' && isDigitChar(l.content[l.pos+1]) {
				kind := scanNumericLiteral(l.content, &l.pos, l.eofPos)
				return l.createToken(kind), tokenMem
			}

			return scanOperatorOrPunctuactorToken(l), tokenMem

		case '/':
			if isSingleLineCommentStart(l.content, l.pos, l.eofPos) {
				scanSingleLineComment(l.content, &l.pos, l.eofPos, l.state)
				continue
			} else if isDelimitedCommentStart(l.content, l.pos, l.eofPos) {
				scanDelimitedComment(l.content, &l.pos, l.eofPos)
				continue
			} else if l.content[l.pos+1] == '=' {
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

func getNameOrDigitTokens(l *LexerScanner, tokenMem []Token) (*Token, []Token) {
	if isNameStart(l.content, l.pos, l.eofPos) {
		scanName(l.content, &l.pos, l.eofPos)
		token := l.createToken(Name)
		tokenText := token.getText(l.content)
		lowerText := strings.ToLower(tokenText)
		if isKeywordOrReservedWordStart(lowerText) {
			token = getKeywordOrReservedWordTokenFromNameToken(token, lowerText, l.content, &l.pos, l.eofPos)
		}
		return token, tokenMem
	} else if isDigitChar(l.content[l.pos]) {
		kind := scanNumericLiteral(l.content, &l.pos, l.eofPos)
		return l.createToken(kind), tokenMem
	}
	l.pos++
	return l.createToken(Unknown), tokenMem
}

func getStringQuoteTokens(l *LexerScanner, tokenMem []Token) (*Token, []Token) {
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
	if token.Kind == YieldKeyword {
		//savedPos := pos;
		//nextToken = scanNextToken();
		//lowerText = strings.ToLower(nextToken.getFullText(text))
		//if (preg_replace('/\s+/', '', strtolower($nextToken->getFullText($text))) == "from") {
		//	token.Kind = YieldFromKeyword;
		//	token.length = *pos - token.fullStart;
		//} else {
		//	*pos = savedPos;
		//}
	}
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

func scanTemplateAndSetTokenValue(l *LexerScanner, tokenMem []Token) []Token {
	startPosition := l.start
	eofPos := l.eofPos
	pos := &l.pos
	fileContent := l.content
	*pos++
	for {
		if *pos >= eofPos {
			// UNTERMINATED, report error
			if len(tokenMem) == 0 {
				tokenMem = append(tokenMem, Token{DoubleQuoteToken, l.fullStart, l.start, l.start - l.fullStart + 1})
				l.fullStart = l.start
				tokenMem = append(tokenMem, Token{EncapsedAndWhitespace, l.fullStart, l.start + 1, *pos - l.fullStart})
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
				if *pos-l.fullStart > 1 {
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
					tokenMem = l.addToMemInPlace(DoubleQuoteToken, startPosition, 1, tokenMem)
					l.start++
					l.fullStart++
				}
				if *pos-startPosition > 2 {
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
					*pos++
					*pos++
					tokenMem = l.addToMem(ArrowToken, *pos, tokenMem)
					if isNameStart(fileContent, *pos, eofPos) {
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

func saveCurlyExpression(lexer *LexerScanner, openToken TokenKind, pos *int, startPosition int, tokenMem []Token) (bool, []Token) {
	if len(tokenMem) == 0 {
		tokenMem = lexer.addToMemInPlace(DoubleQuoteToken, startPosition, 1, tokenMem)
		lexer.start++
		lexer.fullStart++
	}
	if *pos-lexer.start > 2 {
		tokenMem = lexer.addToMem(EncapsedAndWhitespace, *pos, tokenMem)
	}
	openTokenLen := 1
	if openToken == DollarOpenBraceToken {
		openTokenLen = 2
	}
	tokenMem = lexer.addToMemInPlace(openToken, *pos, openTokenLen, tokenMem)
	*pos += openTokenLen
	lexer.fullStart = *pos
	lexer.start = *pos

	for *pos < lexer.eofPos {
		t, tokenMemTmp := lexer.scan(nil)
		lexer.fullStart = *pos
		lexer.start = *pos

		if t.Kind == -1 {
			tokenMem = append(tokenMem, tokenMemTmp...)
			continue
		}

		if t.Kind == Name {
			t.Kind = StringVarname
		}
		tokenMem = append(tokenMem, *t)
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
	for *pos < eofPos {
		charCode := text[*pos]
		if isHexadecimalDigit(charCode) {
			*pos++
			continue
		} else if isDigitChar(charCode) || isNameNonDigitChar(charCode) {
			*pos++
			// REPORT ERROR;
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
		return BinaryLiteralToken
	} else if isHexadecimalLiteralStart(text, *pos, eofPos) {
		*pos += 2
		prevPos = *pos
		isValidHexLiteral := scanHexadecimalLiteral(text, pos, eofPos)
		if prevPos == *pos || !isValidHexLiteral {
			return InvalidHexadecimalLiteral
			// invalid hexadecimal literal
		}
		return HexadecimalLiteralToken
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

			return OctalLiteralToken
		}

		scanDecimalLiteral(text, pos, eofPos)
		return IntegerLiteralToken
	}

	return Unknown
}
