package phpsyms

import "github.com/thanhhaudev/phpsyms/internal/symtype"

// SymbolKind classifies emitted symbols.
type SymbolKind = symtype.SymbolKind

// Kind constants — re-exported from internal/symtype.
const (
	KindClass        = symtype.KindClass
	KindInterface    = symtype.KindInterface
	KindTrait        = symtype.KindTrait
	KindMethod       = symtype.KindMethod
	KindFunction     = symtype.KindFunction
	KindUseImport    = symtype.KindUseImport
	KindStaticCall   = symtype.KindStaticCall
	KindMethodCall   = symtype.KindMethodCall
	KindFunctionCall = symtype.KindFunctionCall
)

// Range is a byte+line span in the source file.
type Range = symtype.Range

// Symbol is a single extracted entity (declaration or call site).
type Symbol = symtype.Symbol
