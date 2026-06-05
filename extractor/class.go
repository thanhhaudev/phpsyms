package extractor

import (
	"github.com/thanhhaudev/phpsyms/internal/symtype"
	"github.com/thanhhaudev/phpsyms/lexer"
)

// ClassDecl matches:
//
//	[final|abstract]* class Name [extends Parent] [implements I1, I2] {
//
// On match, advances past the class header and emits a symtype.Symbol; the
// returned newClass carries the class name forward so MethodDecl can attach
// methods to it.
func ClassDecl(c *Cursor, currentClass string) (symtype.Symbol, bool, string) {
	startPos := c.Pos
	var modifiers []string

	for {
		t := c.Cur()
		if t.Kind != lexer.TokKeyword {
			break
		}
		switch t.Value {
		case "final", "abstract":
			modifiers = append(modifiers, t.Value)
			c.Advance()
			c.SkipTrivia()
			continue
		}
		break
	}

	if c.Cur().Kind != lexer.TokKeyword || c.Cur().Value != "class" {
		c.Pos = startPos
		return symtype.Symbol{}, false, ""
	}
	classKw := c.Cur()
	c.Advance()
	c.SkipTrivia()

	if c.Cur().Kind != lexer.TokIdent {
		// Anonymous class — spike scope does not emit. Phase1 Task 11 will handle.
		c.Pos = startPos
		return symtype.Symbol{}, false, ""
	}
	nameTok := c.Cur()
	c.Advance()
	c.SkipTrivia()

	parent := ""
	if c.Cur().Kind == lexer.TokKeyword && c.Cur().Value == "extends" {
		c.Advance()
		c.SkipTrivia()
		parent = consumeQualifiedName(c)
		c.SkipTrivia()
	}

	var implements []string
	if c.Cur().Kind == lexer.TokKeyword && c.Cur().Value == "implements" {
		c.Advance()
		c.SkipTrivia()
		for {
			name := consumeQualifiedName(c)
			if name == "" {
				break
			}
			implements = append(implements, name)
			c.SkipTrivia()
			if c.Cur().Kind != lexer.TokComma {
				break
			}
			c.Advance()
			c.SkipTrivia()
		}
	}

	return symtype.Symbol{
		Kind:       symtype.KindClass,
		Name:       nameTok.Value,
		Range:      tokenRange(classKw, nameTok),
		Modifiers:  modifiers,
		Parent:     parent,
		Implements: implements,
	}, true, nameTok.Value
}

// consumeQualifiedName reads `Foo\Bar\Baz` style names (with leading \ optional).
func consumeQualifiedName(c *Cursor) string {
	out := ""
	if c.Cur().Kind == lexer.TokBackslash {
		out = "\\"
		c.Advance()
	}
	for {
		if c.Cur().Kind != lexer.TokIdent {
			break
		}
		out += c.Cur().Value
		c.Advance()
		if c.Cur().Kind == lexer.TokBackslash {
			out += "\\"
			c.Advance()
			continue
		}
		break
	}
	return out
}

func tokenRange(start, end lexer.Token) symtype.Range {
	return symtype.Range{
		StartLine: start.StartLine,
		StartCol:  start.StartCol,
		EndLine:   end.StartLine,
		EndCol:    end.StartCol,
		StartByte: start.StartByte,
		EndByte:   end.EndByte,
	}
}
