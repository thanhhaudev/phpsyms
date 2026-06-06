package lexer

// Lex tokenizes src into a Token slice. The final token is always TokEOF.
// Lex MUST NEVER panic; malformed input degrades to TokError or best-effort tokens.
func Lex(filename string, src []byte) []Token {
	// Pre-size token slice: ~1 token per 10 bytes matches observed ratio on PHP corpora.
	// This avoids most append-growth GC pressure while staying proportional to input size.
	initCap := len(src)/10 + 32
	l := &lexer{src: src, line: 1, col: 1, state: stateText, tokens: make([]Token, 0, initCap)}
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
// EndLine/EndCol are taken from l.line/l.col at the moment of the call, which
// must be AFTER all advance() calls for this token — i.e. they reflect the
// position immediately following the last consumed byte.
func (l *lexer) emitAt(kind TokenKind, value string, startLine, startCol, start, end int) {
	l.tokens = append(l.tokens, Token{
		Kind:      kind,
		Value:     value,
		StartLine: startLine,
		StartCol:  startCol,
		EndLine:   l.line,
		EndCol:    l.col,
		StartByte: start,
		EndByte:   end,
	})
}

// emitSinglePunct emits a 1-byte punctuation token. It captures the start
// position, advances 1 byte, then emits so that EndLine/EndCol reflect the
// position AFTER the consumed byte (matching the exclusive-end semantic).
func (l *lexer) emitSinglePunct(kind TokenKind, value string) {
	startLine, startCol := l.line, l.col
	start := l.pos
	l.advance(1)
	l.emitAt(kind, value, startLine, startCol, start, l.pos)
}

// emitTwoBytePunct does the same for 2-byte tokens (e.g. "::", "->").
func (l *lexer) emitTwoBytePunct(kind TokenKind, value string) {
	startLine, startCol := l.line, l.col
	start := l.pos
	l.advance(2)
	l.emitAt(kind, value, startLine, startCol, start, l.pos)
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
		// Fast path: skip runs of whitespace without per-byte advance overhead.
		for l.pos < len(l.src) {
			ch := l.src[l.pos]
			if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
				break
			}
			if ch == '\n' {
				l.line++
				l.col = 1
			} else {
				l.col++
			}
			l.pos++
		}
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
		l.col++
		l.pos++ // skip '$'
		// Inline fast-path: ident bytes never contain newlines.
		for l.pos < len(l.src) && isIdentCont(l.src[l.pos]) {
			l.col++
			l.pos++
		}
		l.emitAt(TokVariable, string(l.src[start:l.pos]), startLine, startCol, start, l.pos)
		return
	}

	if isIdentStart(c) {
		startLine := l.line
		startCol := l.col
		start := l.pos
		// Fast inline: advance past ident bytes without calling advance() to avoid
		// per-byte line/col tracking overhead (column tracking is still correct since
		// identifiers never contain newlines).
		for l.pos < len(l.src) && isIdentCont(l.src[l.pos]) {
			l.col++
			l.pos++
		}
		raw := l.src[start:l.pos]
		// Build lowercase version on the stack for the keyword check, avoiding
		// the extra heap allocation that IsKeyword would incur.
		var lowBuf [16]byte
		isKw := false
		if len(raw) <= 16 {
			for i, b := range raw {
				if b >= 'A' && b <= 'Z' {
					lowBuf[i] = b + ('a' - 'A')
				} else {
					lowBuf[i] = b
				}
			}
			isKw = isKeywordBytes(lowBuf[:len(raw)])
		}
		word := string(raw)
		if isKw {
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
		l.emitSinglePunct(TokLBrace, "{")
	case '}':
		l.emitSinglePunct(TokRBrace, "}")
	case '(':
		l.emitSinglePunct(TokLParen, "(")
	case ')':
		l.emitSinglePunct(TokRParen, ")")
	case '[':
		l.emitSinglePunct(TokLBracket, "[")
	case ']':
		l.emitSinglePunct(TokRBracket, "]")
	case ';':
		l.emitSinglePunct(TokSemi, ";")
	case ',':
		l.emitSinglePunct(TokComma, ",")
	case ':':
		if l.pos+1 < len(l.src) && l.src[l.pos+1] == ':' {
			l.emitTwoBytePunct(TokDoubleColon, "::")
		} else {
			l.emitSinglePunct(TokColon, ":")
		}
	case '-':
		if l.pos+1 < len(l.src) && l.src[l.pos+1] == '>' {
			l.emitTwoBytePunct(TokArrow, "->")
		} else {
			l.emitSinglePunct(TokOther, "-")
		}
	case '\\':
		l.emitSinglePunct(TokBackslash, "\\")
	case '?':
		l.emitSinglePunct(TokQuestion, "?")
	case '=':
		l.emitSinglePunct(TokEquals, "=")
	default:
		l.emitSinglePunct(TokOther, string(c))
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

// isIdentStart reports whether c may begin a PHP identifier.
// c >= 0x80 preserves the existing graceful-degrade behavior for high-byte UTF-8
// (treating any high byte as a valid ident start — same imperfect behavior as before
// but avoids the unicode table lookup for the ASCII fast-path that covers ~99% of PHP).
func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

// isIdentCont reports whether c may continue a PHP identifier.
func isIdentCont(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c >= 0x80
}

func isHeredocEnd(c byte) bool {
	return c == ';' || c == ',' || c == ')' || c == '\n' || c == '\r'
}
