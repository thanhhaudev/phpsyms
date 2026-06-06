package phpsyms

import (
	"github.com/thanhhaudev/phpsyms/extractor"
	"github.com/thanhhaudev/phpsyms/lexer"
)

// Extract tokenizes src and returns symbols in source order.
// v0.1.0 patterns: UseImport, ClassDecl, InterfaceDecl, TraitDecl, MethodDecl, FunctionDecl.
// Call-site patterns (Static/Method/Function calls) come in Task 10.
func Extract(filename string, src []byte) ([]Symbol, error) {
	toks := lexer.Lex(filename, src)
	syms := extractor.Run(toks, []extractor.Pattern{
		extractor.UseImport,
		extractor.ClassDecl,
		extractor.InterfaceDecl,
		extractor.TraitDecl,
		extractor.MethodDecl,
		extractor.FunctionDecl,
	})
	return syms, nil
}
