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
func ClassDecl(c *Cursor, currentClass string) ([]symtype.Symbol, string, bool) {
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
		return nil, "", false
	}
	classKw := c.Cur()
	c.Advance()
	c.SkipTrivia()

	var nameTok lexer.Token
	anonymous := false
	if c.Cur().Kind == lexer.TokIdent {
		nameTok = c.Cur()
		c.Advance()
	} else {
		// Anonymous class: `new class [extends ...] [implements ...] { ... }`.
		// nameTok stays zero; we use classKw position for the Range.
		anonymous = true
		nameTok = classKw
	}
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

	sym := symtype.Symbol{
		Kind:       symtype.KindClass,
		Name:       "",
		Range:      tokenRange(classKw, nameTok),
		Modifiers:  modifiers,
		Parent:     parent,
		Implements: implements,
	}
	if !anonymous {
		sym.Name = nameTok.Value
		return []symtype.Symbol{sym}, nameTok.Value, true
	}
	return []symtype.Symbol{sym}, "<anonymous>", true
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
		EndLine:   end.EndLine,
		EndCol:    end.EndCol,
		StartByte: start.StartByte,
		EndByte:   end.EndByte,
	}
}

// InterfaceDecl matches:
//
//	interface Name [extends I1, I2, I3] {
//
// Multi-extends list is captured in Symbol.Implements since Symbol.Parent is
// single-valued. newClass returned so MethodDecl can attach methods to the
// interface scope.
func InterfaceDecl(c *Cursor, currentClass string) ([]symtype.Symbol, string, bool) {
	startPos := c.Pos
	if c.Cur().Kind != lexer.TokKeyword || c.Cur().Value != "interface" {
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
	c.SkipTrivia()

	var extends []string
	if c.Cur().Kind == lexer.TokKeyword && c.Cur().Value == "extends" {
		c.Advance()
		c.SkipTrivia()
		for {
			name := consumeQualifiedName(c)
			if name == "" {
				break
			}
			extends = append(extends, name)
			c.SkipTrivia()
			if c.Cur().Kind != lexer.TokComma {
				break
			}
			c.Advance()
			c.SkipTrivia()
		}
	}

	return []symtype.Symbol{{
		Kind:       symtype.KindInterface,
		Name:       nameTok.Value,
		Range:      tokenRange(kw, nameTok),
		Implements: extends,
	}}, nameTok.Value, true
}

// TraitDecl matches:
//
//	trait Name {
func TraitDecl(c *Cursor, currentClass string) ([]symtype.Symbol, string, bool) {
	startPos := c.Pos
	if c.Cur().Kind != lexer.TokKeyword || c.Cur().Value != "trait" {
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
		Kind:  symtype.KindTrait,
		Name:  nameTok.Value,
		Range: tokenRange(kw, nameTok),
	}}, nameTok.Value, true
}
