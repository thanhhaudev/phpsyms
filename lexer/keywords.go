package lexer

// keywords is the PHP 7.4 reserved-word set. Case-insensitive in PHP,
// but the lexer normalizes to lower-case before lookup.
var keywords = map[string]bool{
	"abstract":     true,
	"and":          true,
	"array":        true,
	"as":           true,
	"break":        true,
	"callable":     true,
	"case":         true,
	"catch":        true,
	"class":        true,
	"clone":        true,
	"const":        true,
	"continue":     true,
	"declare":      true,
	"default":      true,
	"do":           true,
	"echo":         true,
	"else":         true,
	"elseif":       true,
	"empty":        true,
	"enddeclare":   true,
	"endfor":       true,
	"endforeach":   true,
	"endif":        true,
	"endswitch":    true,
	"endwhile":     true,
	"extends":      true,
	"final":        true,
	"finally":      true,
	"fn":           true,
	"for":          true,
	"foreach":      true,
	"function":     true,
	"global":       true,
	"goto":         true,
	"if":           true,
	"implements":   true,
	"include":      true,
	"include_once": true,
	"instanceof":   true,
	"insteadof":    true,
	"interface":    true,
	"isset":        true,
	"list":         true,
	"namespace":    true,
	"new":          true,
	"or":           true,
	"print":        true,
	"private":      true,
	"protected":    true,
	"public":       true,
	"require":      true,
	"require_once": true,
	"return":       true,
	"self":         true,
	"static":       true,
	"switch":       true,
	"throw":        true,
	"trait":        true,
	"try":          true,
	"unset":        true,
	"use":          true,
	"var":          true,
	"while":        true,
	"xor":          true,
	"yield":        true,
}

// IsKeyword reports whether ident matches a PHP 7.4 reserved word.
// Comparison is case-insensitive per PHP semantics.
func IsKeyword(ident string) bool {
	lower := make([]byte, len(ident))
	for i := 0; i < len(ident); i++ {
		c := ident[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		lower[i] = c
	}
	return keywords[string(lower)]
}
