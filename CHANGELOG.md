# Changelog

## v0.1.0 — 2026-06-06

Initial release. PHP 7.4 baseline.

- State-machine lexer (text/PHP/double-string/heredoc/comment states)
- Patterns: ClassDecl, InterfaceDecl, TraitDecl, MethodDecl, FunctionDecl,
  UseImport (single/group/aliased), StaticCall, MethodCall, FunctionCall
- Anonymous class support
- Public API: `Extract(filename, src) → ([]Symbol, error)`
- Test corpus: ~51 files from laravel/framework + 4 PHP-spec corner cases
- Bench: ≥5000 files/sec on Apple Silicon against the corpus
- Parity floor pinned at the v0.1.0 coverage (139 classes, 843 methods,
  394 use imports, 2911 calls)

## v0.0.1-spike — 2026-06-06

Spike: ClassDecl + MethodDecl only. Validated state-machine approach on
Oneplat samples. Acceptance bars cleared: ≥90% symbol match vs regex
baseline (100% measured), ≥10× speedup vs tree-sitter (~43–170× measured),
zero `$var` leak from string/heredoc interpolation, zero lexer panics.
