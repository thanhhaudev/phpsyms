package extractor

import (
	"github.com/thanhhaudev/phpsyms/internal/symtype"
	"github.com/thanhhaudev/phpsyms/lexer"
)

// StaticCall matches:
//
//	Foo::bar(
//	Foo\Bar::baz(
//	\Root\Foo::bar(
//
// Emits when followed by `(` to avoid matching `Foo::CONST` references.
func StaticCall(c *Cursor, currentClass string) ([]symtype.Symbol, string, bool) {
	startPos := c.Pos
	// Must start at an identifier or backslash (for qualified name).
	if c.Cur().Kind != lexer.TokIdent && c.Cur().Kind != lexer.TokBackslash {
		return nil, "", false
	}
	nameStart := c.Cur()
	cls := consumeQualifiedName(c)
	if cls == "" {
		c.Pos = startPos
		return nil, "", false
	}
	if c.Cur().Kind != lexer.TokDoubleColon {
		c.Pos = startPos
		return nil, "", false
	}
	c.Advance()
	if c.Cur().Kind != lexer.TokIdent {
		c.Pos = startPos
		return nil, "", false
	}
	methodTok := c.Cur()
	// Lookahead for `(` BEFORE consuming the method ident.
	if c.Peek(1).Kind != lexer.TokLParen {
		c.Pos = startPos
		return nil, "", false
	}
	c.Advance() // past method ident; leave `(` for the main loop to handle.
	return []symtype.Symbol{{
		Kind:   symtype.KindStaticCall,
		Name:   methodTok.Value,
		Parent: cls,
		Range:  tokenRange(nameStart, methodTok),
	}}, "", true
}

// MethodCall matches:
//
//	$obj->method(
//	$this->name(
//
// Receiver = variable name with leading $ stripped (e.g. "this", "user").
func MethodCall(c *Cursor, currentClass string) ([]symtype.Symbol, string, bool) {
	startPos := c.Pos
	if c.Cur().Kind != lexer.TokVariable {
		return nil, "", false
	}
	recvTok := c.Cur()
	c.Advance()
	if c.Cur().Kind != lexer.TokArrow {
		c.Pos = startPos
		return nil, "", false
	}
	c.Advance()
	if c.Cur().Kind != lexer.TokIdent {
		c.Pos = startPos
		return nil, "", false
	}
	methodTok := c.Cur()
	if c.Peek(1).Kind != lexer.TokLParen {
		c.Pos = startPos
		return nil, "", false
	}
	c.Advance()
	receiver := recvTok.Value
	if len(receiver) > 0 && receiver[0] == '$' {
		receiver = receiver[1:]
	}
	return []symtype.Symbol{{
		Kind:     symtype.KindMethodCall,
		Name:     methodTok.Value,
		Receiver: receiver,
		Range:    tokenRange(recvTok, methodTok),
	}}, "", true
}

// FunctionCall matches:
//
//	name(
//
// at expression position. Disambiguated from declarations + member access by
// checking the preceding token — skip if previous token is `function`/`class`/
// `interface`/`trait`/`new` (declaration intro) or `->`/`::`/`\` (member access).
func FunctionCall(c *Cursor, currentClass string) ([]symtype.Symbol, string, bool) {
	startPos := c.Pos
	if c.Cur().Kind != lexer.TokIdent {
		return nil, "", false
	}
	nameTok := c.Cur()
	if c.Peek(1).Kind != lexer.TokLParen {
		return nil, "", false
	}
	if startPos > 0 {
		prev := c.Tokens[startPos-1]
		switch prev.Kind {
		case lexer.TokKeyword:
			if prev.Value == "function" || prev.Value == "class" || prev.Value == "interface" || prev.Value == "trait" || prev.Value == "new" {
				return nil, "", false
			}
		case lexer.TokArrow, lexer.TokDoubleColon, lexer.TokBackslash:
			return nil, "", false
		}
	}
	c.Advance()
	return []symtype.Symbol{{
		Kind:  symtype.KindFunctionCall,
		Name:  nameTok.Value,
		Range: tokenRange(nameTok, nameTok),
	}}, "", true
}
