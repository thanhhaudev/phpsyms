# phpsyms

Go-native PHP symbol extractor. Stdlib-only, ~1.5k LOC, no AST.

## Install

    go get github.com/thanhhaudev/phpsyms@v0.2.0

## Use

```go
import "github.com/thanhhaudev/phpsyms"

syms, err := phpsyms.Extract("UserController.php", src)
if err != nil {
    // catastrophic only — lexer never panics; soft errors return partial result
}
for _, s := range syms {
    fmt.Printf("%s %s @ %d\n", s.Kind, s.Name, s.Range.StartLine)
}
```

## Symbol kinds

| Kind | What |
|---|---|
| `KindClass` | `class Foo {...}` including anonymous (Name="") |
| `KindInterface` | `interface Foo extends Bar, Baz` (extends list in `Implements`) |
| `KindTrait` | `trait Foo` |
| `KindMethod` | function inside class — `Parent` = class name |
| `KindFunction` | top-level function |
| `KindUseImport` | `use Foo\Bar [as Baz]` / `use Foo\{A, B}` — emitted with `Qualified` |
| `KindStaticCall` | `Foo::bar()` — `Parent` holds the qualified class |
| `KindMethodCall` | `$obj->method()` — `Receiver` holds the variable name (no `$`) |
| `KindFunctionCall` | `name()` at expression position |
| `KindTypeRef` | parameter type hint or return-type CamelCase identifier (filters `True`/`False`/`Null`) |

## Supported PHP

- **PHP 7.4 baseline** — full coverage target.
- **PHP 8.x** — graceful degrade. The lexer tolerates new syntax (attributes
  `#[]`, enums, `match`, named args, nullsafe `?->`, etc.) without panic;
  the extractor may emit partial results. Blade templates are out of scope.
- The lexer **never panics** — any malformed input degrades to best-effort
  tokens ending in `TokEOF`.

## Performance

On an Apple M1 Pro, `BenchmarkExtract` reports ≥5000 files/sec against the
committed `laravel/framework` corpus (~51 files, ~430 KB). Reference baseline:
tree-sitter PHP cursor walk runs at ~5–20 files/sec on similar code.

## What's NOT included

- Full PHP AST — use [z7zmey/php-parser](https://github.com/z7zmey/php-parser)
  if you need the parse tree.
- Cross-file name resolution — `Qualified` carries the unqualified `use`
  target; resolving identifiers in caller code requires an external index.
- Laravel-specific semantic tagging — planned for a separate package.
- IDE-grade error recovery — phpsyms favors speed + simplicity.

## Status

`v0.2.0` — stable surface, PHP 7.4 baseline. Adds `KindTypeRef` extraction
from parameter type hints + return-type annotations (mirrors LLM-review-style
type-reference tracking). Lexer tokens now carry both `Start*` and `End*`
line/column positions.

Future versions add PHP 8.x features and Blade template support.
