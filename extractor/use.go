package extractor

import (
	"github.com/thanhhaudev/phpsyms/internal/symtype"
	"github.com/thanhhaudev/phpsyms/lexer"
)

// UseImport matches the various `use` forms at file scope:
//
//	use Foo\Bar;
//	use Foo\Bar as Baz;
//	use Foo\{A, B as C};
//
// Gated on currentClass == "" to avoid matching `use TraitName;` inside class
// body (trait composition — different semantics, deferred to a later task).
func UseImport(c *Cursor, currentClass string) ([]symtype.Symbol, string, bool) {
	if currentClass != "" {
		return nil, "", false
	}
	startPos := c.Pos
	if c.Cur().Kind != lexer.TokKeyword || c.Cur().Value != "use" {
		return nil, "", false
	}
	kw := c.Cur()
	c.Advance()
	c.SkipTrivia()

	// Optional `function` or `const` prefix — rare; skip the keyword.
	if c.Cur().Kind == lexer.TokKeyword && (c.Cur().Value == "function" || c.Cur().Value == "const") {
		c.Advance()
		c.SkipTrivia()
	}

	prefix := consumeQualifiedName(c)
	if prefix == "" {
		c.Pos = startPos
		return nil, "", false
	}
	c.SkipTrivia()

	// Group form: use App\Http\{Request, Response, X as Y};
	if c.Cur().Kind == lexer.TokLBrace {
		c.Advance()
		c.SkipTrivia()
		var syms []symtype.Symbol
		for {
			member := consumeQualifiedName(c)
			if member == "" {
				break
			}
			full := prefix
			if len(full) == 0 || full[len(full)-1] != '\\' {
				full += "\\"
			}
			full += member
			c.SkipTrivia()
			if c.Cur().Kind == lexer.TokKeyword && c.Cur().Value == "as" {
				c.Advance()
				c.SkipTrivia()
				if c.Cur().Kind == lexer.TokIdent {
					c.Advance()
				}
			}
			syms = append(syms, symtype.Symbol{
				Kind:      symtype.KindUseImport,
				Qualified: full,
				Range:     tokenRange(kw, c.Tokens[c.Pos-1]),
			})
			c.SkipTrivia()
			if c.Cur().Kind != lexer.TokComma {
				break
			}
			c.Advance()
			c.SkipTrivia()
		}
		return syms, "", true
	}

	// Single form, possibly aliased.
	rangeEnd := c.Tokens[c.Pos-1]
	if c.Cur().Kind == lexer.TokKeyword && c.Cur().Value == "as" {
		c.Advance()
		c.SkipTrivia()
		if c.Cur().Kind == lexer.TokIdent {
			c.Advance()
		}
	}
	return []symtype.Symbol{{
		Kind:      symtype.KindUseImport,
		Qualified: prefix,
		Range:     tokenRange(kw, rangeEnd),
	}}, "", true
}
