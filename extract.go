package phpsyms

import (
	"github.com/thanhhaudev/phpsyms/extractor"
	"github.com/thanhhaudev/phpsyms/lexer"
)

// defaultPatterns is the canonical ordered pattern set for Extract.
// Package-level to avoid re-allocating the slice on every call.
var defaultPatterns = []extractor.Pattern{
	// Declarations first to consume name tokens.
	extractor.UseImport,
	extractor.ClassDecl,
	extractor.InterfaceDecl,
	extractor.TraitDecl,
	extractor.MethodDecl,
	extractor.FunctionDecl,
	// Calls last (catch-all).
	extractor.StaticCall,
	extractor.MethodCall,
	extractor.FunctionCall,
}

// Extract tokenizes src and returns symbols in source order.
// v0.1.0 patterns: UseImport, ClassDecl, InterfaceDecl, TraitDecl, MethodDecl, FunctionDecl.
// v0.2.0 patterns: StaticCall, MethodCall, FunctionCall.
func Extract(filename string, src []byte) ([]Symbol, error) {
	toks := lexer.Lex(filename, src)
	syms := extractor.Run(toks, defaultPatterns)
	return syms, nil
}
