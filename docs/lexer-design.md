# phpsyms lexer — design walkthrough

This document explains the state-machine lexer that powers phpsyms. The
audience is contributors and anyone curious about how a stdlib-only Go
lexer turns PHP source into a token stream that downstream pattern
matchers can walk.

Source files referenced throughout:
- `lexer/lexer.go` — the state-machine driver
- `lexer/state.go` — the six-state enum
- `lexer/token.go` — `Token` + `TokenKind` types
- `lexer/keywords.go` — the PHP 7.4 reserved-word table

---

## 1. What a lexer is

A lexer (also called tokenizer or scanner) consumes a raw source string
and emits a sequence of typed tokens — the smallest units of meaning.

```
Input  (string):    <?php class Foo {}
                      │ lexer
                      ▼
Output (tokens):    [TokOpenTag "<?php",
                     TokKeyword "class",
                     TokIdent   "Foo",
                     TokLBrace,
                     TokRBrace,
                     TokEOF]
```

Each token carries:
- **Kind** — the category (keyword, identifier, punctuation, string, ...).
- **Value** — the raw text (used for keyword disambiguation and for
  carrying the identifier name).
- **Position** — line + column + byte offset of both the first and last
  byte of the token (since phpsyms v0.2.0 the lexer reports `EndLine` and
  `EndCol` too, not just `Start*`).

The output is intentionally flat — no tree, no parse hierarchy. The next
pass (`extractor/`) walks this flat stream with pattern matchers, which
is what gives phpsyms its speed (~5000 files/sec) and small footprint
(~1.5k LOC).

---

## 2. Why state matters

The naive temptation is to write a lexer as one big `switch` on the
current byte. That fails the moment you realize the same byte means
different things in different contexts.

### Example: the dollar sign `$`

```php
<?php $user = "Hello $name";
```

The byte `$` appears twice and means two different things:
- At position 6 (`$user`): the start of a **variable** — should emit
  `TokVariable "$user"`.
- Inside the string at position 19 (`$name`): part of the **string body**
  — should NOT emit a variable; the entire `"Hello $name"` is one
  `TokString` whose value is `Hello $name`.

A stateless `switch '$': emitVariable()` would leak `$name` out of the
string and produce a broken token stream, which downstream matchers
would then misinterpret as a real variable reference. That would
manifest as false-positive findings in code review.

### Example: the word `class`

```php
<?php class Foo {} ?><html>Lorem class CSS docs</html>
```

The four bytes `class` appear twice. The first occurrence is the PHP
`class` keyword (`TokKeyword`). The second is just text inside an HTML
section and should be folded into a single `TokInlineHTML` token along
with everything else after `?>`. Same byte sequence, opposite intent.

### Generalization

Any non-trivial source language has contexts where the same input bytes
mean different things:
- Inside vs. outside `<?php ... ?>`
- Inside vs. outside a string literal
- Inside vs. outside a comment
- Inside vs. outside a heredoc body

The lexer needs to track **which context it is in** so it can apply the
right tokenization rules. That tracking is called **state**.

---

## 3. State machines, briefly

A state machine is a small structured way to encode "I'm in mode X; if
I see byte Y, switch to mode Z and emit token T."

Concretely in phpsyms (`lexer/state.go`):

```go
type state int

const (
    stateText state = iota // outside <?php
    statePHP               // inside <?php ... ?>
    stateDoubleString      // inside "..." (interpolation possible)
    stateHeredoc           // inside <<<LABEL ... LABEL
    stateLineComment       // // ... or # ...
    stateBlockComment      // /* ... */
)
```

The driver loop is just:

```go
for l.pos < len(l.src) {
    switch l.state {
    case stateText:         l.lexText()
    case statePHP:          l.lexPHP()
    case stateDoubleString: l.lexDoubleString()
    case stateHeredoc:      l.lexHeredoc()
    case stateLineComment:  l.lexLineComment()
    case stateBlockComment: l.lexBlockComment()
    }
}
l.emit(TokEOF, ...)
```

Each `lex*` handler reads bytes within its own rules. When it detects a
transition trigger (e.g., `lexPHP` sees a `"`), it emits any pending
token, advances past the trigger, and sets `l.state` to the new mode.
The driver picks up the new state on the next loop iteration.

State machines are a workhorse in compiler engineering. Every PHP parser,
every JavaScript engine, every regex implementation has one. phpsyms
just uses it explicitly rather than burying it in nested conditions.

---

## 4. The six phpsyms states

Each state describes one tokenization mode with its own rules and exit
triggers.

### 4.1 `stateText` — outside `<?php`

Used at the top of a file before any PHP tag, and after a `?>` until the
next `<?php`.

Behavior:
- Read bytes until either `<?php` is seen or EOF is reached.
- Emit the accumulated bytes as one `TokInlineHTML` (skipped entirely if
  zero-length).
- On `<?php`: emit `TokOpenTag "<?php"` and transition to `statePHP`.

Why a dedicated state: PHP source files often start with raw HTML or
template prose. We don't want every `<` and `>` in HTML to be parsed
as PHP punctuation.

### 4.2 `statePHP` — the main mode

Used everywhere inside `<?php ... ?>` blocks. This is the busiest
handler in `lexer.go`.

Behavior:
- Skip whitespace silently.
- Recognize identifiers and disambiguate keyword vs ident via
  `IsKeyword` (case-insensitive lookup in `keywords.go`).
- Recognize variables (`$ident`). A bare `$` followed by non-identifier
  is left as `TokOther`.
- Recognize numbers (rough — accepts `0-9 . x X a-f A-F _`; numeric
  literal correctness isn't critical for symbol extraction).
- Recognize string starts:
  - `'` → enter inline single-string handler (no interpolation, stays in
    `statePHP`).
  - `"` → transition to `stateDoubleString`.
  - `<<<` → transition to `stateHeredoc` via `lexHeredocStart` which
    parses the label.
- Recognize comments:
  - `//` or `#` → transition to `stateLineComment`.
  - `/*` → transition to `stateBlockComment`.
- Recognize punctuation: emit one of `TokLBrace`, `TokRBrace`,
  `TokLParen`, `TokRParen`, `TokLBracket`, `TokRBracket`, `TokSemi`,
  `TokComma`, `TokDoubleColon` (`::`), `TokArrow` (`->`), `TokBackslash`,
  `TokColon`, `TokQuestion`, `TokEquals`. Anything else falls through to
  `TokOther`.
- On `?>`: emit `TokCloseTag` and transition to `stateText`.

### 4.3 `stateDoubleString` — inside `"..."`

Used between the opening `"` and the closing `"` of a double-quoted
string.

Behavior:
- Read bytes until the closing `"`.
- Handle escapes: `\\X` skips two bytes so `\"` doesn't terminate the
  string.
- **Crucially**, do NOT split out `$var` interpolation as a separate
  token. The entire span (including any `$name`, `{$obj->prop}`,
  `${name}`) is emitted as one `TokString` with the raw body as `Value`.
  Downstream consumers can re-tokenize the body themselves if they
  need to, but for symbol extraction we want the body opaque.
- On `"`: emit the accumulated `TokString` and transition back to
  `statePHP`.

This is the state where most "is `$name` a variable?" bugs would
originate if state were ignored.

### 4.4 `stateHeredoc` — inside `<<<LABEL`

PHP heredocs and nowdocs.

```php
$x = <<<EOT
hello $name
EOT;
```

Behavior:
- Read bytes until the closing label is seen at the start of a line.
  PHP 7.3+ allows whitespace indentation before the label; the lexer
  accepts that.
- Termination: scan forward at column 1, optionally skip leading
  whitespace, then check whether the next `len(label)` bytes equal the
  recorded heredoc label AND are followed by one of `;`, `,`, `)`,
  `\n`, `\r`, or EOF.
- On termination: emit the body as one `TokString` and transition back
  to `statePHP`.
- Like `stateDoubleString`, `$var` interpolation stays inside the body.
- Nowdoc (single-quoted label `<<<'EOT'`) is handled identically at the
  body level — `heredocIsNow` is recorded but the body is still emitted
  as one opaque `TokString`. The semantic difference (nowdoc skips
  interpolation in real PHP runtime) is invisible here because we
  never expand interpolation either way.

### 4.5 `stateLineComment` — `//` or `#`

Behavior:
- Read bytes until `\n` or EOF.
- Emit the comment body as one `TokComment` (the `\n` itself is not
  consumed — `statePHP` handles it on return).
- Transition back to `statePHP`.

### 4.6 `stateBlockComment` — `/* ... */`

Behavior:
- Read bytes until `*/` or EOF.
- Emit the comment body as one `TokComment`. Unterminated comments
  emit what was read so far, then EOF closes the file.
- Transition back to `statePHP`.

Block comments are a separate state because they span multiple lines
and can contain `//` or `#` or any other punctuation that would
otherwise trigger a state change in `statePHP`.

---

## 5. A traced example

Let's trace `<?php $x = "hi $name";` byte by byte.

```
Pos  Byte   State                Action / token
═════════════════════════════════════════════════════════════════════
 0    '<'   stateText            scan for <?php trigger
 1    '?'   stateText            (still scanning)
 2    'p'   stateText            (still scanning)
 3    'h'   stateText            (still scanning)
 4    'p'   stateText            match! emit TokOpenTag "<?php"
                                 → state := statePHP
 5    ' '   statePHP             whitespace, skip
 6    '$'   statePHP             $ + ident start → variable scan
 7    'x'   statePHP             continue var
 8    ' '   statePHP             end of var, emit TokVariable "$x"
                                 then whitespace, skip
 9    '='   statePHP             emit TokEquals "="
10    ' '   statePHP             whitespace, skip
11    '"'   statePHP             string start → state := stateDoubleString
12    'h'   stateDoubleString    accumulate body
13    'i'   stateDoubleString    accumulate body
14    ' '   stateDoubleString    accumulate body
15    '$'   stateDoubleString    accumulate body — KEY MOMENT
16    'n'   stateDoubleString    accumulate body
17    'a'   stateDoubleString    accumulate body
18    'm'   stateDoubleString    accumulate body
19    'e'   stateDoubleString    accumulate body
20    '"'   stateDoubleString    close! emit TokString "hi $name"
                                 → state := statePHP
21    ';'   statePHP             emit TokSemi ";"
       EOF                        emit TokEOF
```

The token stream produced:

```
[TokOpenTag "<?php",
 TokVariable "$x",
 TokEquals "=",
 TokString "hi $name",
 TokSemi ";",
 TokEOF]
```

Six tokens. Note how `$name` at position 15-19 stayed glued to the
string body precisely because `stateDoubleString` has different rules
than `statePHP`. A stateless lexer would have produced something like:

```
[..., TokVariable "$x", TokEquals, TokOther '"',
      TokIdent "hi", TokVariable "$name", TokOther '"', TokSemi]
```

— which would falsely promote `$name` to a real variable in the symbol
graph.

---

## 6. Companion pass: pattern matchers

States produce tokens. Tokens drive matchers. The `extractor/` package
walks the flat token stream with a tiny `Cursor` and tries each
registered pattern in order:

- `UseImport` — `use Foo\Bar [as Baz]` or `use Foo\{A, B}`.
- `ClassDecl`, `InterfaceDecl`, `TraitDecl` — declarations with modifier
  prefixes, extends/implements lists.
- `MethodDecl`, `FunctionDecl` — gated on whether we're inside a class
  body (tracked via a brace-depth counter in `extractor.Run`).
- `StaticCall`, `MethodCall`, `FunctionCall` — call-site patterns,
  disambiguated by the preceding token (e.g., `name(` after `function`
  keyword is a declaration, not a call).
- `collectTypeRefs` — runs as part of `MethodDecl` / `FunctionDecl` to
  emit `KindTypeRef` for CamelCase identifiers in parameter type hints
  and return types.

Each matcher returns either `(symbols, true)` or `(nil, false)`. On
false the cursor rewinds to the position it started at, so the next
matcher gets a clean attempt. This try-rewind pattern is how phpsyms
copes without backtracking lookahead — the lexer never emits
ambiguous tokens, so matchers only need a few bytes of context.

---

## 7. Why six states and not three, not ten?

Design questions worth examining:

### Could we collapse states?

- **Single-quoted strings** don't have interpolation, so they're scanned
  inline within `statePHP` rather than getting their own state. Decision:
  fewer states wins because the parse logic is trivial enough to inline.
- **Heredoc vs. nowdoc** share the same body-scanning logic at this
  level. We track `heredocIsNow` in the lexer struct for completeness
  but don't expose it because downstream pattern matchers don't care.
  One state covers both.
- **Line comment `//` vs. `#`** could share a state with block
  comments, but the termination conditions are different (`\n` vs.
  `*/`). Inlining the difference inside one handler would mean
  branching constantly on the open trigger; two states with one rule
  each is cleaner.

### Could we add more states?

- **PHP 8.x attributes `#[...]`** would conflict with the `#` line-
  comment trigger. v0.1.0 doesn't support 8.x; if we add it, expect a
  new state (or a tweak to `statePHP` to peek `#[` before falling into
  `stateLineComment`).
- **Backtick exec syntax** `` `ls -la` `` is an exotic PHP corner; not
  worth a state until someone actually needs it.
- **Multi-mode strings** (e.g., separating interpolation runs into
  their own tokens) — would require ~3 new states. We explicitly do
  NOT do this because the extractor doesn't need interpolation expanded.

The six chosen states are the minimum that correctly handles PHP 7.4
constructs without producing ambiguous tokens that the downstream
extractor would have to disambiguate by re-tokenizing.

---

## 8. Adding a new state (contributor guide)

If you need to extend phpsyms to handle a new tokenization mode (PHP
8.x attributes, backtick exec, an exotic doc-comment annotation, ...),
the moves are:

1. **Add the state constant** in `lexer/state.go`. Pick a descriptive
   name in `lower_camelCase`.

2. **Add a handler method** on `lexer` in `lexer/lexer.go` — `lex<Name>`.
   It should read bytes until its termination trigger, emit any token
   it accumulates, and transition `l.state` back (usually to `statePHP`).

3. **Wire the dispatcher** in `run()` — add a `case` to the switch.

4. **Wire the transition trigger** — somewhere in `lexPHP` (or whichever
   parent state can enter the new one), detect the entry sequence,
   set `l.state = newState`, and return so the driver loop picks up
   the new state.

5. **Test heavily**. Add tests for:
   - The happy path (proper termination).
   - Unterminated input (EOF mid-state must not panic).
   - Interaction with other states (e.g., does `?>` work inside the
     new state if applicable?).
   - Fuzz seed corpus addition in `FuzzLex`.

6. **Document the new state** in this file (extend §4).

---

## 9. Common pitfalls and edge cases

Things future contributors are likely to trip on:

- **`l.col` resets to 1 on `\n` BEFORE `l.pos` advances**. This is
  important for heredoc termination, which checks `l.col == 1` to
  detect "we're at the start of a line." If you change `advance()`,
  preserve this invariant.

- **Single-byte vs. multi-byte token emit semantics differ**. Multi-byte
  tokens (idents, strings, numbers, comments) capture `startLine` and
  `startCol` BEFORE the scan loop, then call `emitAt` after the scan
  with the captured start. Single-byte punctuation captures inline
  (start = current position) and advances after emit. If you add a new
  token, follow the existing pattern in `lexer.go` precisely.

- **`emitAt` records `EndLine`/`EndCol` from `l.line`/`l.col` at call
  time**. Callers MUST ensure they call `emitAt` AFTER advancing the
  cursor past the last byte of the token. Calling `emitAt` before
  the final `advance` produces wrong end positions.

- **The lexer must never panic**. Every loop has bounds checks; every
  state must handle EOF gracefully. The `FuzzLex` corpus is here to
  catch this. If you write a new state and you can produce ANY input
  that panics or hangs, that's a regression — fuzz first, then add a
  seed.

- **String / heredoc bodies are opaque**. If you need interpolation
  expanded (e.g., for security analysis), do it in a second pass on
  the `Value` field of the `TokString`. Don't re-architect the lexer
  to split it — you'd lose the speed advantage and introduce a
  whole new class of "did the `}` close the brace or the interpolation"
  bugs.

---

## 10. Further reading

- [Rob Pike's "Lexical Scanning in Go" talk (2011)](https://www.youtube.com/watch?v=HxaD_trXwRE)
  — the canonical reference for state-machine lexers in Go. phpsyms
  uses a simpler driver (no goroutines, no channels) but the conceptual
  model is the same.
- The PHP language reference grammar — `Zend/zend_language_scanner.l`
  in the official php-src — is a re2c lexer that we essentially
  reimplement in Go. Reading it is the best way to discover edge cases
  we haven't hit yet.
- For a different style entirely, see `z7zmey/php-parser` which builds
  a full AST. Useful as a reference when you wonder whether phpsyms is
  underconstrained — usually we are, intentionally.
