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

var pos int = 0
var endOfFilePos = 0
var fullStart = 0
var start = 0
var lexState LexerState
var fileContent []rune
var tokenMem []Token

func GetTokens(text string) []Token {
	var tokens []Token
	tokenMem = nil
	fileContent = []rune(text)
	endOfFilePos = len(fileContent)
	pos = 0
	start = 0
	lexState = LexStateHtmlSection
	token := scan()
	for token.Kind != EndOfFileToken {
		if token.Kind == -1 {
			for i := 0; i < len(tokenMem); i++ {
				tokens = append(tokens, tokenMem[i])
			}
			tokenMem = nil
		} else {
			tokens = append(tokens, token)
			pos = token.fullStart + token.length
		}
		token = scan()
	}
	tokens = append(tokens, token)
	return tokens
}

func DebugTokens(text string) {
	tokens := GetTokens(text)
	for _, token := range tokens {
		b, _ := json.MarshalIndent(token.getShortForm([]rune(text)), "", "    ")
		os.Stdout.Write(b)
		println("")
	}
}

func isScriptStartTag(text []rune, pos int, endOfFilePos int) bool {

	if text[pos] != chCode_lessThan {
		return false
	}
	if pos+5 > endOfFilePos {
		return false
	}

	start := strings.ToLower(string(text[pos : 5+pos]))
	end := text[pos+5]

	if start == "<?php" && (end == '\n' || end == '\r' || end == ' ' || end == '\t') ||
		string(text[pos:pos+3]) == "<?=" {
		return true
	}
	return false
}

func scan() Token {
	fullStart = pos
	var current Token
	for {
		start = pos
		// handling end of file
		if pos >= endOfFilePos {
			if lexState != LexStateHtmlSection {
				current = Token{EndOfFileToken, fullStart, start, pos - fullStart}
			} else {
				current = Token{InlineHtml, fullStart, fullStart, pos - fullStart}
			}
			lexState = LexStateScriptSection
			if current.Kind == InlineHtml && pos-fullStart == 0 {
				continue
			}
			return current
		}

		if lexState == LexStateHtmlSection {
			// Keep scanning until we hit a script section start tag
			if !isScriptStartTag(fileContent, pos, endOfFilePos) {
				pos++
				continue
			}
			lexState = LexStateScriptSection

			if pos-fullStart == 0 {
				continue
			}
			return Token{InlineHtml, fullStart, fullStart, pos - fullStart}
		}

		quoteStart := false

		charCode := fileContent[pos]

		//println("char code: ", fileContent[pos:pos+1], charCode)

		switch charCode {
		case chCode_hash:
			scanSingleLineComment(fileContent, &pos, endOfFilePos)
			continue
		case chCode_space, chCode_tab, chCode_return, chCode_newline:
			pos++
			continue
		case chCode_dot: // ..., .=, . // TODO also applies to floating point literals
			if isDigitChar(fileContent[pos+1]) {
				kind := scanNumericLiteral(fileContent, &pos, endOfFilePos)
				return Token{kind, fullStart, start, pos - fullStart}
			}
			// Otherwise fall through to compounds
		case chCode_lessThan, // <=>, <=, <<=, <<, < // TODO heredoc and nowdoc
			chCode_equals,      // ===, ==, =
			chCode_greaterThan, // >>=, >>, >=, >
			chCode_asterisk,    // **=, **, *=, *
			chCode_exclamation, // !==, !=, !

			// Potential 2-char compound
			chCode_plus,      // +=, ++, +
			chCode_minus,     // -= , --, ->, -
			chCode_percent,   // %=, %
			chCode_caret,     // ^=, ^
			chCode_bar,       // |=, ||, |
			chCode_ampersand, // &=, &&, &
			chCode_question,  // ??, ?, end-tag

			chCode_colon, // : (TODO should this actually be treated as compound?)
			chCode_comma, // , (TODO should this actually be treated as compound?)

			// Non-compound
			chCode_at, // @
			chCode_openBracket,
			chCode_closeBracket,
			chCode_openParen,
			chCode_closeParen,
			chCode_openBrace,
			chCode_closeBrace,
			chCode_semicolon,
			chCode_tilde,
			chCode_backslash:
			// TODO this can be made more performant, but we're going for simple/correct first.
			// TODO
			var tokenKind TokenKind
			for tokenEnd := 6; tokenEnd >= 0; tokenEnd-- {
				if pos+tokenEnd >= endOfFilePos {
					continue
				}
				// TODO get rid of strtolower for perf reasons
				textSubstring := strings.ToLower(string(fileContent[pos : pos+tokenEnd+1]))
				if isOperatorOrPunctuator(textSubstring) {
					tokenKind = OPERATORS_AND_PUNCTUATORS[textSubstring]
					if tokenKind == ScriptSectionStartTag {
						if lexState == LexStateScriptSectionParsed {
							continue
						}
						lexState = LexStateScriptSectionParsed
					}
					pos += tokenEnd + 1
					if tokenKind == ScriptSectionEndTag {
						lexState = LexStateHtmlSection
					}

					return Token{tokenKind, fullStart, start, pos - fullStart}
				}
			}
			//panic("Unknown token Kind");
			return Token{Unknown, fullStart, start, pos - fullStart}

		case chCode_slash:
			if isSingleLineCommentStart(fileContent, pos, endOfFilePos) {
				scanSingleLineComment(fileContent, &pos, endOfFilePos)
				continue
			} else if isDelimitedCommentStart(fileContent, pos, endOfFilePos) {
				scanDelimitedComment(fileContent, &pos, endOfFilePos)
				continue
			} else if fileContent[pos+1] == chCode_equals {
				pos += 2
				return Token{SlashEqualsToken, fullStart, start, pos - fullStart}
			}
			pos++
			return Token{SlashToken, fullStart, start, pos - fullStart}

		case chCode_dollar:
			pos++
			if isNameStart(fileContent, pos, endOfFilePos) {
				scanName(fileContent, &pos, endOfFilePos)
				return Token{VariableName, fullStart, start, pos - fullStart}
			}
			return Token{DollarToken, fullStart, start, pos - fullStart}
		default:

			if charCode == chCode_doubleQuote || charCode == chCode_singleQuote || charCode == chCodeb || charCode == chCodeB {
				if charCode == chCode_doubleQuote || charCode == chCode_singleQuote {
					quoteStart = true
				}

				if fileContent[pos] == chCode_singleQuote ||
					fileContent[pos] == chCode_doubleQuote ||
					(pos+1 < endOfFilePos && (fileContent[pos+1] == chCode_singleQuote || fileContent[pos+1] == chCode_doubleQuote)) {
					if quoteStart == false {
						pos++
					}
					if fileContent[pos] == chCode_doubleQuote {
						scanTemplateAndSetTokenValue(fileContent, &pos, endOfFilePos, false)
						return Token{-1, fullStart, start, pos - fullStart}
					}
					pos++
					if scanStringLiteral(fileContent, &pos, endOfFilePos) {
						return Token{StringLiteralToken, fullStart, start, pos - fullStart}
					}
					return Token{EncapsedAndWhitespace, fullStart, start, pos - fullStart}
				}
			}

			if isNameStart(fileContent, pos, endOfFilePos) {
				scanName(fileContent, &pos, endOfFilePos)
				token := Token{Name, fullStart, start, pos - fullStart}
				tokenText := token.getText(fileContent)
				lowerText := strings.ToLower(tokenText)
				if isKeywordOrReservedWordStart(lowerText) {
					token = getKeywordOrReservedWordTokenFromNameToken(&token, lowerText, fileContent, &pos, endOfFilePos)
				}
				return token
			} else if isDigitChar(fileContent[pos]) {
				kind := scanNumericLiteral(fileContent, &pos, endOfFilePos)
				return Token{kind, fullStart, start, pos - fullStart}
			}
			pos++
			return Token{Unknown, fullStart, start, pos - fullStart}
		}
	}
}

func getKeywordOrReservedWordTokenFromNameToken(token *Token, lowerKeywordStart string, text []rune, pos *int, endOfFilePos int) Token {

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
	return *token
}

func isDigitChar(at rune) bool {
	return at >= chCode_0 &&
		at <= chCode_9
}

func isKeywordOrReservedWordStart(text string) bool {
	_, ok := KEYWORDS[text]
	_, ok2 := RESERVED_WORDS[text]
	return ok || ok2
}

func scanStringLiteral(text []rune, pos *int, endOfFilePos int) bool {
	// TODO validate with multiple character sets
	isTerminated := false
	for *pos < endOfFilePos {
		if isSingleQuoteEscapeSequence(text, *pos) {
			*pos += 2
			continue
		} else if text[*pos] == chCode_singleQuote {
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
func scanDelimitedComment(text []rune, pos *int, endOfFilePos int) {
	for *pos < endOfFilePos {
		if *pos+1 < endOfFilePos && text[*pos] == chCode_asterisk && text[*pos+1] == chCode_slash {
			*pos += 2
			return
		}
		*pos++
	}

}

func scanName(text []rune, pos *int, endOfFilePos int) {
	for *pos < endOfFilePos {
		charCode := text[*pos]
		if isNameNonDigitChar(charCode) || isDigitChar(charCode) {
			*pos++
			continue
		}
		return
	}

}

func saveMemToken(kind TokenKind, pos int) {
	tokenMem = append(tokenMem, Token{kind, fullStart, start, pos - fullStart})
	fullStart = pos
	start = pos
}

func scanTemplateAndSetTokenValue(text []rune, pos *int, endOfFilePos int, isRescan bool) {
	*pos++
	for {
		if *pos >= endOfFilePos {
			// UNTERMINATED, report error
			if len(tokenMem) == 0 {
				tokenMem = append(tokenMem, Token{DoubleQuoteToken, fullStart, start, start - fullStart + 1})
				fullStart = start
				tokenMem = append(tokenMem, Token{EncapsedAndWhitespace, fullStart, start + 1, *pos - fullStart})
				return
			} else {
				saveMemToken(UnterminatedTemplateStringEnd, *pos)
				return
			}
		}

		char := text[*pos]

		if char == chCode_doubleQuote {

			if len(tokenMem) == 0 {
				*pos++
				saveMemToken(StringLiteralToken, *pos)
				return
				//return NoSubstitutionTemplateLiteral
			} else {
				if *pos-fullStart > 1 {
					saveMemToken(EncapsedAndWhitespace, *pos)
				}
				*pos++
				saveMemToken(DoubleQuoteToken, *pos)
				return
			}
		}

		if char == '$' {
			if isNameStart(fileContent, *pos+1, endOfFilePos) {
				if len(tokenMem) == 0 {
					saveMemToken(DoubleQuoteToken, *pos)
				}
				if *pos-start > 2 {
					saveMemToken(EncapsedAndWhitespace, *pos)
				}
				*pos++
				scanName(fileContent, pos, endOfFilePos)
				saveMemToken(VariableName, *pos)

				if *pos < endOfFilePos && fileContent[*pos] == '[' {
					*pos++
					saveMemToken(OpenBracketToken, *pos)
					if isDigitChar(fileContent[*pos]) {
						*pos++
						scanName(fileContent, pos, endOfFilePos)
						saveMemToken(IntegerLiteralToken, *pos)
					} else if isNameStart(fileContent, *pos, endOfFilePos) {
						// var name index
						*pos++
						scanName(fileContent, pos, endOfFilePos)
						saveMemToken(Name, *pos)
					}
					if fileContent[*pos] == ']' {
						*pos++
						saveMemToken(CloseBracketToken, *pos)
					} else {
						// todo error!
					}
				} else if *pos+1 < endOfFilePos && fileContent[*pos] == '-' && fileContent[*pos+1] == '>' {
					*pos++
					*pos++
					saveMemToken(ArrowToken, *pos)
					if isNameStart(fileContent, *pos, endOfFilePos) {
						// var name index
						*pos++
						scanName(fileContent, pos, endOfFilePos)
						saveMemToken(Name, *pos)
					}
				}

				continue
			} else if *pos+1 < endOfFilePos && fileContent[*pos+1] == '{' {
				// curly
				if saveCurlyExpression(DollarOpenBraceToken, pos) {
					return
				}
			}
		}

		if char == '{' {
			if *pos+1 < endOfFilePos && fileContent[*pos+1] == '$' {
				if saveCurlyExpression(OpenBraceDollarToken, pos) {
					return
				}
			}
		}

		// Escape character
		if char == chCode_backslash {
			*pos++
			scanDqEscapeSequence(text, pos, endOfFilePos)
			continue
		}

		*pos++
	}

	// TODO throw error
	return
}

func saveCurlyExpression(openToken TokenKind, pos *int) bool {
	if len(tokenMem) == 0 {
		saveMemToken(DoubleQuoteToken, *pos)
		*pos++
	}
	if *pos-start > 2 {
		saveMemToken(EncapsedAndWhitespace, *pos)
	}
	if openToken == DollarOpenBraceToken {
		*pos++
	}
	saveMemToken(openToken, *pos)
	t := scan()
	fullStart = *pos
	start = *pos
	if t.Kind == Name {
		t.Kind = StringVarname
	}
	tokenMem = append(tokenMem, t)
	if t.Kind == ScriptSectionEndTag {
		return true
	}
	if *pos < endOfFilePos && fileContent[*pos] == '}' {
		saveMemToken(CloseBraceToken, *pos+1)
	}
	return false
}

func scanDqEscapeSequence(text []rune, pos *int, endOfFilePos int) {
	if *pos >= endOfFilePos {
		// ERROR
		return
	}
	char := text[*pos]
	switch char {
	// dq-simple-escape-sequence
	case chCode_doubleQuote,
		chCode_backslash,
		chCode_dollar,
		chCodee,
		chCodef,
		chCoder,
		chCodet,
		chCodev:
		*pos++
		return

		// dq-hexadecimal-escape-sequence
	case chCodex,
		chCodeX:
		*pos++
		for i := 0; i < 2; i++ {
			if isHexadecimalDigit(text[*pos]) {
				*pos++
			}
		}
		return

		// dq-unicode-escape-sequence
	case chCodeu:
		*pos++
		if text[*pos] == chCode_openBrace {
			scanHexadecimalLiteral(text, pos, endOfFilePos)
			if text[*pos] == chCode_closeBrace {
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

func scanOctalLiteral(text []rune, pos *int, endOfFilePos int) bool {
	isValid := true
	for *pos < endOfFilePos {
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

func scanDecimalLiteral(text []rune, pos *int, endOfFilePos int) {
	for *pos < endOfFilePos {
		charCode := text[*pos]
		if isDigitChar(charCode) {
			*pos++
			continue
		}
		return
	}
}
func scanSingleLineComment(text []rune, pos *int, endOfFilePos int) {
	for *pos < endOfFilePos {
		if isNewLineChar(text[*pos]) || isScriptEndTag(text, *pos) {
			return
		}
		*pos++
	}
}
func isSingleLineCommentStart(text []rune, pos int, endOfFilePos int) bool {
	return pos+1 < endOfFilePos && text[pos] == chCode_slash && text[pos+1] == chCode_slash
}

func isSingleQuoteEscapeSequence(text []rune, pos int) bool {
	return text[pos] == chCode_backslash &&
		(chCode_singleQuote == text[pos+1] || chCode_backslash == text[pos+1])
}

func isScriptEndTag(text []rune, pos int) bool {
	if lexState != LexStateScriptSection && text[pos] == chCode_question && text[pos+1] == chCode_greaterThan {
		return true
	}
	return false
}

func isNewLineChar(charCode rune) bool {
	return charCode == chCode_newline || charCode == chCode_return
}

func isNonzeroDigitChar(charCode rune) bool {
	return charCode >= chCode_1 &&
		charCode <= chCode_9
}

func isOctalDigitChar(charCode rune) bool {
	return charCode >= chCode_0 &&
		charCode <= chCode_7
}

func isBinaryDigitChar(charCode rune) bool {
	return charCode == chCode_0 ||
		charCode == chCode_1
}

func isHexadecimalDigit(charCode rune) bool {
	// 0  1  2  3  4  5  6  7  8  9
	// a  b  c  d  e  f
	// A  B  C  D  E  F
	return charCode >= chCode_0 && charCode <= chCode_9 || charCode >= chCodea && charCode <= chCodef || charCode >= chCodeA && charCode <= chCodeF
}

func isNameNonDigitChar(charCode rune) bool {
	return isNonDigitChar(charCode) || isValidNameUnicodeChar(charCode)
}

func isNonDigitChar(charCode rune) bool {
	return (charCode >= chCodea && charCode <= chCodez) ||
		(charCode >= chCodeA && charCode <= chCodeZ) ||
		charCode == chCode_underscore
}

func isValidNameUnicodeChar(charCode rune) bool {
	return unicode.IsLetter(charCode)
}

func scanHexadecimalLiteral(text []rune, pos *int, endOfFilePos int) bool {
	isValid := true
	for *pos < endOfFilePos {
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
func scanFloatingPointLiteral(text []rune, pos *int, endOfFilePos int) bool {
	hasDot := false
	var expStart int = -1
	hasSign := false
	for *pos < endOfFilePos {
		char := text[*pos]
		if isDigitChar(char) {
			*pos++
			continue
		} else if char == chCode_dot {
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

func scanBinaryLiteral(text []rune, pos *int, endOfFilePos int) bool {
	isValid := true
	for *pos < endOfFilePos {
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

func isNameStart(text []rune, pos int, endOfFilePos int) bool {
	return pos < endOfFilePos && isNameNonDigitChar(text[pos])
}
func isDelimitedCommentStart(text []rune, pos int, endOfFilePos int) bool {
	return pos+1 < endOfFilePos && text[pos] == chCode_slash && text[pos+1] == chCode_asterisk
}

func isHexadecimalLiteralStart(text []rune, pos int, endOfFilePos int) bool {
	// 0x  0X
	return text[pos] == '0' &&
		(text[pos+1] == 'x' || text[pos+1] == 'X')
}
func isBinaryLiteralStart(text []rune, pos int, endOfFilePos int) bool {
	return text[pos] == '0' && (text[pos+1] == 'b' || text[pos+1] == 'B')
}

func scanNumericLiteral(text []rune, pos *int, endOfFilePos int) TokenKind {
	var prevPos int

	if isBinaryLiteralStart(text, *pos, endOfFilePos) {
		*pos += 2
		prevPos = *pos
		isValidBinaryLiteral := scanBinaryLiteral(text, pos, endOfFilePos)
		if prevPos == *pos || !isValidBinaryLiteral {
			// invalid binary literal
			return InvalidBinaryLiteral
		}
		return BinaryLiteralToken
	} else if isHexadecimalLiteralStart(text, *pos, endOfFilePos) {
		*pos += 2
		prevPos = *pos
		isValidHexLiteral := scanHexadecimalLiteral(text, pos, endOfFilePos)
		if prevPos == *pos || !isValidHexLiteral {
			return InvalidHexadecimalLiteral
			// invalid hexadecimal literal
		}
		return HexadecimalLiteralToken
	} else if isDigitChar(text[*pos]) || text[*pos] == chCode_dot {
		// TODO throw error if there is no number past the dot.
		prevPos = *pos
		isValidFloatingLiteral := scanFloatingPointLiteral(text, pos, endOfFilePos)
		if isValidFloatingLiteral {
			return FloatingLiteralToken
		}

		// Reset, try scanning octal literal
		*pos = prevPos
		if text[*pos] == '0' {
			isValidOctalLiteral := scanOctalLiteral(text, pos, endOfFilePos)

			// Check that it's not a 0 decimal literal
			if *pos == prevPos+1 {
				return IntegerLiteralToken
			}

			if !isValidOctalLiteral {
				return InvalidOctalLiteralToken
			}

			return OctalLiteralToken
		}

		scanDecimalLiteral(text, pos, endOfFilePos)
		return IntegerLiteralToken
	}

	return Unknown
}

func isOperatorOrPunctuator(text string) bool {
	_, ok := OPERATORS_AND_PUNCTUATORS[string(text)]
	return ok
}
