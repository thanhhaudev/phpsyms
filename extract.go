package phpsyms

import (
	"github.com/thanhhaudev/phpsyms/extractor"
	"github.com/thanhhaudev/phpsyms/internal/symtype"
	"github.com/thanhhaudev/phpsyms/lexer"
)

// Extract tokenizes src and returns symbols in source order.
// Spike build registers only ClassDecl + MethodDecl; v0.1.0 adds the rest.
func Extract(filename string, src []byte) ([]Symbol, error) {
	toks := lexer.Lex(filename, src)
	syms := extractor.Run(toks, []func(c *extractor.Cursor, current string) (symtype.Symbol, bool, string){
		extractor.ClassDecl,
		extractor.MethodDecl,
	})
	return syms, nil
}
