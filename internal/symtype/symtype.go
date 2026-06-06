// Package symtype defines the public symbol types shared between the extractor
// and the root phpsyms package, avoiding import cycles.
package symtype

// SymbolKind classifies emitted symbols.
type SymbolKind int

const (
	KindClass SymbolKind = iota
	KindInterface
	KindTrait
	KindMethod
	KindFunction
	KindUseImport
	KindStaticCall
	KindMethodCall
	KindFunctionCall
	KindTypeRef // parameter type hints + return type annotations (CamelCase identifiers)
)

// Range is a byte+line span in the source file.
type Range struct {
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
	StartByte int
	EndByte   int
}

// Symbol is a single extracted entity (declaration or call site).
type Symbol struct {
	Kind       SymbolKind
	Name       string
	Qualified  string
	Range      Range
	Modifiers  []string
	Parent     string
	Implements []string
	Receiver   string
	Hint       map[string]string
}
