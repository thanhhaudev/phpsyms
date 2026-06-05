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

// Run walks all tokens and applies registered patterns. The order matters —
// pattern attempts are tried in the order added. Patterns return
// (sym, ok, newClass) — when ok is true the symbol is appended; when newClass
// is non-empty it becomes the current class scope for subsequent MethodDecl
// matches. On no-match the cursor is rewound to startPos and the next pattern
// is tried.
func Run(toks []lexer.Token, patterns []func(c *Cursor, currentClass string) (symtype.Symbol, bool, string)) []symtype.Symbol {
	var out []symtype.Symbol
	c := &Cursor{Tokens: toks}
	currentClass := ""
	for !c.Done() {
		c.SkipTrivia()
		if c.Done() {
			break
		}
		matched := false
		for _, p := range patterns {
			startPos := c.Pos
			sym, ok, newClass := p(c, currentClass)
			if ok {
				out = append(out, sym)
				if newClass != "" {
					currentClass = newClass
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
