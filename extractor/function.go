package extractor

import (
	"github.com/thanhhaudev/phpsyms/internal/symtype"
	"github.com/thanhhaudev/phpsyms/lexer"
)

// FunctionDecl matches top-level (NOT inside class) function declarations:
//
//	function name(...) [: RetType] {
//
// Methods are handled by MethodDecl; this is gated on currentClass == "".
func FunctionDecl(c *Cursor, currentClass string) ([]symtype.Symbol, string, bool) {
	if currentClass != "" {
		return nil, "", false
	}
	startPos := c.Pos
	if c.Cur().Kind != lexer.TokKeyword || c.Cur().Value != "function" {
		return nil, "", false
	}
	kw := c.Cur()
	c.Advance()
	c.SkipTrivia()
	if c.Cur().Kind != lexer.TokIdent {
		c.Pos = startPos
		return nil, "", false
	}
	nameTok := c.Cur()
	c.Advance()
	return []symtype.Symbol{{
		Kind:  symtype.KindFunction,
		Name:  nameTok.Value,
		Range: tokenRange(kw, nameTok),
	}}, "", true
}
