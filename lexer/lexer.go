package lexer

import "unicode"

// Lex tokenizes src into a Token slice. The final token is always TokEOF.
// Lex MUST NEVER panic; malformed input degrades to TokError or best-effort tokens.
func Lex(filename string, src []byte) []Token {
	l := &lexer{src: src, line: 1, col: 1, state: stateText}
	l.run()
	return l.tokens
}

type lexer struct {
	src    []byte
	pos    int
	line   int
	col    int
	state  state
	tokens []Token

	heredocLabel string
	heredocIsNow bool // nowdoc (single-quoted label) — skip interpolation
}

func (l *lexer) run() {
	for l.pos < len(l.src) {
		switch l.state {
		case stateText:
			l.lexText()
		case statePHP:
			l.lexPHP()
		case stateDoubleString:
			l.lexDoubleString()
		case stateHeredoc:
			l.lexHeredoc()
		case stateLineComment:
			l.lexLineComment()
		case stateBlockComment:
			l.lexBlockComment()
		}
	}
	l.emitAt(TokEOF, "", l.line, l.col, l.pos, l.pos)
}

// emitAt appends a token using explicitly provided start position metadata.
// Use this for all tokens — it captures StartLine/StartCol at the token's first byte,
// not at the post-advance position.
func (l *lexer) emitAt(kind TokenKind, value string, startLine, startCol, start, end int) {
	l.tokens = append(l.tokens, Token{
		Kind:      kind,
		Value:     value,
		StartLine: startLine,
		StartCol:  startCol,
		StartByte: start,
		EndByte:   end,
	})
}

func (l *lexer) advance(n int) {
	for i := 0; i < n && l.pos < len(l.src); i++ {
		if l.src[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
}

func (l *lexer) lexText() {
	startLine := l.line
	startCol := l.col
	start := l.pos
	for l.pos < len(l.src) {
		if l.pos+5 <= len(l.src) && string(l.src[l.pos:l.pos+5]) == "<?php" {
			if start < l.pos {
				l.emitAt(TokInlineHTML, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
			}
			tagStart := l.pos
			tagLine := l.line
			tagCol := l.col
			l.advance(5)
			l.emitAt(TokOpenTag, "<?php", tagLine, tagCol, tagStart, l.pos)
			l.state = statePHP
			return
		}
		l.advance(1)
	}
	if start < l.pos {
		l.emitAt(TokInlineHTML, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
	}
}

func (l *lexer) lexPHP() {
	if l.pos+2 <= len(l.src) && string(l.src[l.pos:l.pos+2]) == "?>" {
		tagLine := l.line
		tagCol := l.col
		tagStart := l.pos
		l.advance(2)
		l.emitAt(TokCloseTag, "?>", tagLine, tagCol, tagStart, l.pos)
		l.state = stateText
		return
	}

	c := l.src[l.pos]

	if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
		l.advance(1)
		return
	}

	if c == '/' && l.pos+1 < len(l.src) {
		next := l.src[l.pos+1]
		if next == '/' {
			l.state = stateLineComment
			return
		}
		if next == '*' {
			l.state = stateBlockComment
			return
		}
	}
	if c == '#' {
		l.state = stateLineComment
		return
	}

	if c == '\'' {
		l.lexSingleString()
		return
	}
	if c == '"' {
		l.advance(1)
		l.state = stateDoubleString
		return
	}

	if c == '<' && l.pos+3 < len(l.src) && string(l.src[l.pos:l.pos+3]) == "<<<" {
		l.lexHeredocStart()
		return
	}

	if c == '$' && l.pos+1 < len(l.src) && isIdentStart(l.src[l.pos+1]) {
		startLine := l.line
		startCol := l.col
		start := l.pos
		l.advance(1)
		for l.pos < len(l.src) && isIdentCont(l.src[l.pos]) {
			l.advance(1)
		}
		l.emitAt(TokVariable, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
		return
	}

	if isIdentStart(c) {
		startLine := l.line
		startCol := l.col
		start := l.pos
		for l.pos < len(l.src) && isIdentCont(l.src[l.pos]) {
			l.advance(1)
		}
		word := string(l.src[start:l.pos])
		if IsKeyword(word) {
			l.emitAt(TokKeyword, word, startLine, startCol, start, l.pos)
		} else {
			l.emitAt(TokIdent, word, startLine, startCol, start, l.pos)
		}
		return
	}

	if c >= '0' && c <= '9' {
		startLine := l.line
		startCol := l.col
		start := l.pos
		for l.pos < len(l.src) && (l.src[l.pos] >= '0' && l.src[l.pos] <= '9' || l.src[l.pos] == '.' || l.src[l.pos] == 'x' || l.src[l.pos] == 'X' || (l.src[l.pos] >= 'a' && l.src[l.pos] <= 'f') || (l.src[l.pos] >= 'A' && l.src[l.pos] <= 'F') || l.src[l.pos] == '_') {
			l.advance(1)
		}
		l.emitAt(TokNumber, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
		return
	}

	switch c {
	case '{':
		l.emitAt(TokLBrace, "{", l.line, l.col, l.pos, l.pos+1)
		l.advance(1)
	case '}':
		l.emitAt(TokRBrace, "}", l.line, l.col, l.pos, l.pos+1)
		l.advance(1)
	case '(':
		l.emitAt(TokLParen, "(", l.line, l.col, l.pos, l.pos+1)
		l.advance(1)
	case ')':
		l.emitAt(TokRParen, ")", l.line, l.col, l.pos, l.pos+1)
		l.advance(1)
	case '[':
		l.emitAt(TokLBracket, "[", l.line, l.col, l.pos, l.pos+1)
		l.advance(1)
	case ']':
		l.emitAt(TokRBracket, "]", l.line, l.col, l.pos, l.pos+1)
		l.advance(1)
	case ';':
		l.emitAt(TokSemi, ";", l.line, l.col, l.pos, l.pos+1)
		l.advance(1)
	case ',':
		l.emitAt(TokComma, ",", l.line, l.col, l.pos, l.pos+1)
		l.advance(1)
	case ':':
		if l.pos+1 < len(l.src) && l.src[l.pos+1] == ':' {
			l.emitAt(TokDoubleColon, "::", l.line, l.col, l.pos, l.pos+2)
			l.advance(2)
		} else {
			l.emitAt(TokColon, ":", l.line, l.col, l.pos, l.pos+1)
			l.advance(1)
		}
	case '-':
		if l.pos+1 < len(l.src) && l.src[l.pos+1] == '>' {
			l.emitAt(TokArrow, "->", l.line, l.col, l.pos, l.pos+2)
			l.advance(2)
		} else {
			l.emitAt(TokOther, "-", l.line, l.col, l.pos, l.pos+1)
			l.advance(1)
		}
	case '\\':
		l.emitAt(TokBackslash, "\\", l.line, l.col, l.pos, l.pos+1)
		l.advance(1)
	case '?':
		l.emitAt(TokQuestion, "?", l.line, l.col, l.pos, l.pos+1)
		l.advance(1)
	case '=':
		l.emitAt(TokEquals, "=", l.line, l.col, l.pos, l.pos+1)
		l.advance(1)
	default:
		l.emitAt(TokOther, string(c), l.line, l.col, l.pos, l.pos+1)
		l.advance(1)
	}
}

func (l *lexer) lexSingleString() {
	startLine := l.line
	startCol := l.col
	l.advance(1) // opening '
	start := l.pos
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '\\' && l.pos+1 < len(l.src) {
			l.advance(2)
			continue
		}
		if c == '\'' {
			l.emitAt(TokString, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
			l.advance(1) // closing '
			return
		}
		l.advance(1)
	}
	// Unterminated — emit what we have.
	l.emitAt(TokString, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
}

func (l *lexer) lexDoubleString() {
	startLine := l.line
	startCol := l.col
	start := l.pos
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '\\' && l.pos+1 < len(l.src) {
			l.advance(2)
			continue
		}
		if c == '"' {
			l.emitAt(TokString, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
			l.advance(1)
			l.state = statePHP
			return
		}
		l.advance(1)
	}
	l.emitAt(TokString, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
	l.state = statePHP
}

func (l *lexer) lexHeredocStart() {
	l.advance(3)
	for l.pos < len(l.src) && (l.src[l.pos] == ' ' || l.src[l.pos] == '\t') {
		l.advance(1)
	}
	l.heredocIsNow = false
	if l.pos < len(l.src) && l.src[l.pos] == '\'' {
		l.heredocIsNow = true
		l.advance(1)
	} else if l.pos < len(l.src) && l.src[l.pos] == '"' {
		l.advance(1)
	}
	labelStart := l.pos
	for l.pos < len(l.src) && isIdentCont(l.src[l.pos]) {
		l.advance(1)
	}
	l.heredocLabel = string(l.src[labelStart:l.pos])
	if l.pos < len(l.src) && (l.src[l.pos] == '\'' || l.src[l.pos] == '"') {
		l.advance(1)
	}
	for l.pos < len(l.src) && l.src[l.pos] != '\n' {
		l.advance(1)
	}
	if l.pos < len(l.src) {
		l.advance(1)
	}
	l.state = stateHeredoc
}

func (l *lexer) lexHeredoc() {
	startLine := l.line
	startCol := l.col
	start := l.pos
	for l.pos < len(l.src) {
		if l.col == 1 {
			scan := l.pos
			for scan < len(l.src) && (l.src[scan] == ' ' || l.src[scan] == '\t') {
				scan++
			}
			label := l.heredocLabel
			if scan+len(label) <= len(l.src) && string(l.src[scan:scan+len(label)]) == label {
				end := scan + len(label)
				if end == len(l.src) || isHeredocEnd(l.src[end]) {
					if start < l.pos {
						l.emitAt(TokString, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
					}
					l.advance(end - l.pos)
					l.state = statePHP
					return
				}
			}
		}
		l.advance(1)
	}
	if start < l.pos {
		l.emitAt(TokString, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
	}
	l.state = statePHP
}

func (l *lexer) lexLineComment() {
	startLine := l.line
	startCol := l.col
	start := l.pos
	for l.pos < len(l.src) && l.src[l.pos] != '\n' {
		l.advance(1)
	}
	l.emitAt(TokComment, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
	l.state = statePHP
}

func (l *lexer) lexBlockComment() {
	startLine := l.line
	startCol := l.col
	start := l.pos
	l.advance(2) // skip /*
	for l.pos < len(l.src) {
		if l.pos+1 < len(l.src) && l.src[l.pos] == '*' && l.src[l.pos+1] == '/' {
			l.advance(2)
			l.emitAt(TokComment, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
			l.state = statePHP
			return
		}
		l.advance(1)
	}
	l.emitAt(TokComment, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
	l.state = statePHP
}

func isIdentStart(c byte) bool {
	return c == '_' || unicode.IsLetter(rune(c))
}

func isIdentCont(c byte) bool {
	return c == '_' || unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c))
}

func isHeredocEnd(c byte) bool {
	return c == ';' || c == ',' || c == ')' || c == '\n' || c == '\r'
}
