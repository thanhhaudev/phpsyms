// Package phpsyms is a Go-native PHP symbol extractor.
//
// It parses PHP source into a token stream using a state-machine lexer,
// then walks tokens with pattern matchers to emit Symbol records for
// declarations and calls (Class, Interface, Trait, Method, Function,
// UseImport, StaticCall, MethodCall, FunctionCall).
//
// phpsyms is NOT a full PHP parser. It optimizes for fast, accurate
// symbol extraction without building an AST. PHP 7.4 baseline; later
// versions are tolerated (lexer never panics) but may emit partial
// results.
//
// Public entry point: Extract(filename string, src []byte) ([]Symbol, error).
package phpsyms
