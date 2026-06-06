package extractor

import (
	"github.com/thanhhaudev/phpsyms/internal/symtype"
	"github.com/thanhhaudev/phpsyms/lexer"
)

// MethodDecl matches:
//
//	[public|protected|private|static|abstract|final]* function name(...) [: RetType] {
//
// Only emits when inside a class scope (currentClass != "").
func MethodDecl(c *Cursor, currentClass string) ([]symtype.Symbol, string, bool) {
	if currentClass == "" {
		return nil, "", false
	}
	startPos := c.Pos
	var modifiers []string

	for {
		t := c.Cur()
		if t.Kind != lexer.TokKeyword {
			break
		}
		switch t.Value {
		case "public", "protected", "private", "static", "abstract", "final":
			modifiers = append(modifiers, t.Value)
			c.Advance()
			c.SkipTrivia()
			continue
		}
		break
	}

	if c.Cur().Kind != lexer.TokKeyword || c.Cur().Value != "function" {
		c.Pos = startPos
		return nil, "", false
	}
	fnKw := c.Cur()
	c.Advance()
	c.SkipTrivia()

	if c.Cur().Kind != lexer.TokIdent {
		c.Pos = startPos
		return nil, "", false
	}
	nameTok := c.Cur()
	c.Advance()

	return []symtype.Symbol{{
		Kind:      symtype.KindMethod,
		Name:      nameTok.Value,
		Range:     tokenRange(fnKw, nameTok),
		Modifiers: modifiers,
		Parent:    currentClass,
	}}, "", true
}
