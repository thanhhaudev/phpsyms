# Test data

## laravel-framework/

~50 .php files copied from https://github.com/laravel/framework (MIT license).
Used for parity testing against tree-sitter (Task 11) and as a stable benchmark
corpus (Task 12).

Reference commit: 8c231d0 Refresh unchanged compiled Blade views (#60401)
Clone date: 2026-06-06

## php-spec/

Hand-crafted PHP 7.4 corner cases. Each file targets a specific lexer or
extractor risk:

- `heredoc.php` — heredoc, nowdoc, indented closing label (PHP 7.3+)
- `interpolation.php` — `$var`, `{$obj->prop}`, `${name}`, `{$arr['k']}` forms
- `null_coalesce.php` — nullable type hints, `??` chains
- `anonymous_class.php` — `new class {...}` (Task 11 scope)

Add a new file → add an entry above explaining what risk it targets.
