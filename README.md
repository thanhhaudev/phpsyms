# phpsyms

Go-native PHP symbol extractor. Stdlib-only.

## Status

Pre-release. Spike in progress. See `docs/superpowers/specs/` in the kizunax-plugin-cc repo for full design.

## Use

```go
syms, err := phpsyms.Extract("UserController.php", src)
```

Returns `[]Symbol` in source order.

## Supported PHP

- PHP 7.4 baseline (full coverage target for v0.1.0)
- PHP 8.x graceful degrade (no panic, partial results)
