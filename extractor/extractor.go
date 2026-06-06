// Package extractor matches token-stream patterns against the lexer output
// and emits phpsyms.Symbol records. Each pattern is a small function that
// scans forward from the current position and either advances + emits or
// returns false without consuming.
package extractor

import (
	"github.com/thanhhaudev/phpsyms/internal/symtype"
	"github.com/thanhhaudev/phpsyms/lexer"
)

// Cursor walks a token slice with rewind support.
type Cursor struct {
	Tokens []lexer.Token
	Pos    int
}

func (c *Cursor) Peek(offset int) lexer.Token {
	i := c.Pos + offset
	if i < 0 || i >= len(c.Tokens) {
		return lexer.Token{Kind: lexer.TokEOF}
	}
	return c.Tokens[i]
}

func (c *Cursor) Cur() lexer.Token { return c.Peek(0) }
func (c *Cursor) Done() bool       { return c.Cur().Kind == lexer.TokEOF }
func (c *Cursor) Advance()         { c.Pos++ }

// SkipTrivia advances past comment tokens (the lexer discards whitespace into
// no tokens, but comments emit; skip them for pattern logic).
func (c *Cursor) SkipTrivia() {
	for c.Pos < len(c.Tokens) && c.Cur().Kind == lexer.TokComment {
		c.Pos++
	}
}

// Pattern matches starting at c.Cur. On match, returns one or more symbols,
// the new "current class" scope (or empty to keep current), and ok=true.
// Patterns MUST advance the cursor past the consumed tokens on match;
// MUST leave the cursor at startPos on no-match (handled by caller for safety).
type Pattern func(c *Cursor, currentClass string) (syms []symtype.Symbol, newClass string, ok bool)

// Run walks all tokens and applies registered patterns. The order matters —
// pattern attempts are tried in the order added. Patterns return
// (syms, newClass, ok) — when ok is true the symbols are appended; when
// newClass is non-empty it becomes the current class scope for subsequent
// MethodDecl matches. On no-match the cursor is rewound to startPos and the
// next pattern is tried.
//
// Brace depth is tracked to clear currentClass when execution exits the
// class body, so top-level functions after a class are not misattributed as
// methods.
func Run(toks []lexer.Token, patterns []Pattern) []symtype.Symbol {
	var out []symtype.Symbol
	c := &Cursor{Tokens: toks}
	currentClass := ""
	classDepth := 0 // brace depth at which currentClass was opened
	braceDepth := 0
	for !c.Done() {
		c.SkipTrivia()
		if c.Done() {
			break
		}

		// Track brace depth for class scope exit. Update BEFORE pattern dispatch
		// so a pattern emitted at this token sees the correct scope.
		switch c.Cur().Kind {
		case lexer.TokLBrace:
			braceDepth++
		case lexer.TokRBrace:
			braceDepth--
			if currentClass != "" && braceDepth < classDepth {
				currentClass = ""
				classDepth = 0
			}
		}

		matched := false
		for _, p := range patterns {
			startPos := c.Pos
			syms, newClass, ok := p(c, currentClass)
			if ok {
				out = append(out, syms...)
				if newClass != "" {
					currentClass = newClass
					// Class body brace not yet entered; record depth AT WHICH
					// the body will sit (braceDepth + 1).
					classDepth = braceDepth + 1
				}
				matched = true
				break
			}
			c.Pos = startPos
		}
		if !matched {
			c.Advance()
		}
	}
	return out
}
