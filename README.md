# phpsyms

Go-native PHP symbol extractor. Stdlib-only, ~1.5k LOC, no AST.

## Install

    go get github.com/thanhhaudev/phpsyms@v0.2.1

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

## How it works

phpsyms is not a parser. It runs in two passes:

1. **State-machine lexer** (`lexer/`) tokenizes PHP source into a flat
   `[]Token` stream. Six states: text (outside `<?php`), PHP, double-quoted
   string, heredoc/nowdoc, line comment, block comment. Whitespace is
   discarded; `$var` interpolation stays inside string tokens (never leaks
   as `TokVariable`). The lexer never panics — malformed input degrades to
   best-effort tokens ending in `TokEOF`.

2. **Token-stream pattern matchers** (`extractor/`) walk the token stream
   with a small `Cursor`. Each matcher (ClassDecl, MethodDecl, UseImport,
   StaticCall, ...) inspects the cursor head and either advances + emits a
   `Symbol`, or returns `false` without consuming. A simple brace-depth
   counter tracks when execution leaves a class body so methods are
   attached to the right `Parent`.

No AST, no parse tree, no error recovery beyond "skip the token, try the
next pattern". This is what makes it fast (~5000 files/sec) and small
(~1.5k LOC for full PHP 7.4 coverage plus TypeRef extraction).

## Example

**Input** (`UserController.php`):

```php
<?php
namespace App\Http\Controllers;

use App\Models\User;
use App\Http\Requests\StoreUserRequest;

class UserController extends Controller implements Authenticatable
{
    public function show(?User $user): JsonResponse
    {
        Cache::get('key');
        return $this->respond($user);
    }
}
```

**Output** (`phpsyms.Extract(...)`):

```
KindUseImport     App\Models\User                 line 4
KindUseImport     App\Http\Requests\StoreUserRequest  line 5
KindClass         UserController                   line 7
                    Parent=Controller
                    Implements=[Authenticatable]
                    Modifiers=[]
KindMethod        show                             line 9
                    Parent=UserController
                    Modifiers=[public]
KindTypeRef       User                             line 9   (param hint)
KindTypeRef       JsonResponse                     line 9   (return type)
KindStaticCall    get                              line 11
                    Parent=Cache
KindMethodCall    respond                          line 12
                    Receiver=this
```

Note how phpsyms handles:
- `extends` + `implements` on the class, with multi-value `Implements` slice
- Nullable type hint `?User` and return type `JsonResponse` → emitted as
  separate `KindTypeRef` entries (true/false/null are filtered out)
- Static call `Cache::get(...)` → `KindStaticCall` with `Parent=Cache`
- Method call `$this->respond(...)` → `Receiver` strips the leading `$`
- Anonymous classes (`new class {}`) get emitted with `Name=""` and a
  sentinel scope marker so nested methods still resolve to the right parent

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

## License

[MIT](LICENSE).
