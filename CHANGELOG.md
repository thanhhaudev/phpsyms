# Changelog

## v0.2.0 — 2026-06-06

- New `KindTypeRef` symbol kind: emitted from function/method parameter type
  hints + return-type annotations. Uses CamelCase regex `[A-Z][A-Za-z0-9_]*`
  and filters PHP pseudo-constants `True`/`False`/`Null`. Mirrors the SymTypeRef
  emission of the tree-sitter PHP walker in llmreviewkit.
- Lexer `Token` struct gains `EndLine` + `EndCol` (exclusive end positions).
- `tokenRange` now uses `end.EndLine` / `end.EndCol` so `Symbol.Range` covers
  the full span of the last token, not just its start position.
- Cursor now carries the source bytes (`Source []byte`) so patterns can
  slice annotation text for regex extraction.
- Parity floor on the laravel-framework corpus updated: typeRefs ≥ 244.

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
