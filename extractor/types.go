package extractor

import (
	"github.com/thanhhaudev/phpsyms/internal/symtype"
	"github.com/thanhhaudev/phpsyms/lexer"
)

// collectTypeRefs scans the parameter list and optional return type annotation
// that follow a function/method name token. It emits one KindTypeRef symbol
// per CamelCase type name found (mirroring llmreviewkit's tree-sitter
// SymTypeRef emission).
//
// The cursor must be positioned immediately after the function name token when
// this is called. On return the cursor is positioned at (or past) the opening
// `{` / `;` that ends the signature, ready for the Run loop to track brace
// depth correctly. If the source slice is nil or the parameter list is absent,
// returns nil without advancing.
func collectTypeRefs(c *Cursor) []symtype.Symbol {
	src := c.Source
	if src == nil {
		return nil
	}

	c.SkipTrivia()
	if c.Cur().Kind != lexer.TokLParen {
		return nil
	}

	// Capture parameter list byte range (between the outer `(` and `)`).
	// Also record the opening `(` token so we can attach its line position to
	// every param TypeRef — this is the "param-list anchor" line number.
	lparenTok := c.Cur()
	paramsStart := lparenTok.StartByte + 1 // byte after `(`
	paramsEnd := -1
	returnTypeStart := -1
	returnTypeEnd := -1
	var returnTypeTok lexer.Token // first non-trivia token of the return type

	c.Advance()
	depth := 1
	for !c.Done() && depth > 0 {
		switch c.Cur().Kind {
		case lexer.TokLParen:
			depth++
		case lexer.TokRParen:
			depth--
			if depth == 0 {
				paramsEnd = c.Cur().StartByte // byte before `)`
			}
		}
		c.Advance()
	}
	// c.Cur is now the token AFTER `)`.

	// Optional return type: `: Type` before `{` or `;`.
	c.SkipTrivia()
	if c.Cur().Kind == lexer.TokColon {
		c.Advance()
		c.SkipTrivia()
		returnTypeTok = c.Cur() // first real token of the return type annotation
		returnTypeStart = returnTypeTok.StartByte
		// Scan until `{` (method/function body) or `;` (abstract/interface method).
		for !c.Done() {
			k := c.Cur().Kind
			if k == lexer.TokLBrace || k == lexer.TokSemi {
				returnTypeEnd = c.Cur().StartByte
				break
			}
			c.Advance()
		}
		// If we reached EOF without seeing `{` or `;`, use the last token's end.
		if returnTypeEnd < 0 && c.Pos > 0 {
			returnTypeEnd = c.Tokens[c.Pos-1].EndByte
		}
	}

	// Build TypeRef symbols.
	var out []symtype.Symbol

	if paramsStart >= 0 && paramsEnd > paramsStart {
		paramText := string(src[paramsStart:paramsEnd])
		for _, typeName := range extractTypeNames(paramText) {
			out = append(out, symtype.Symbol{
				Kind: symtype.KindTypeRef,
				Name: typeName,
				Range: symtype.Range{
					StartLine: lparenTok.StartLine,
					StartCol:  lparenTok.StartCol,
					StartByte: paramsStart,
					EndByte:   paramsEnd,
				},
			})
		}
	}

	if returnTypeStart >= 0 && returnTypeEnd > returnTypeStart {
		retText := string(src[returnTypeStart:returnTypeEnd])
		for _, typeName := range extractTypeNames(retText) {
			out = append(out, symtype.Symbol{
				Kind: symtype.KindTypeRef,
				Name: typeName,
				Range: symtype.Range{
					StartLine: returnTypeTok.StartLine,
					StartCol:  returnTypeTok.StartCol,
					StartByte: returnTypeStart,
					EndByte:   returnTypeEnd,
				},
			})
		}
	}

	return out
}
