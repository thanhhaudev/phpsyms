package extractor_test

import (
	"testing"

	"github.com/thanhhaudev/phpsyms"
)

func TestExtract_BasicClass(t *testing.T) {
	src := []byte(`<?php
final class UserController extends Controller implements Authenticatable {
    public function index() {}
    protected static function helper() {}
}`)
	syms, err := phpsyms.Extract("UserController.php", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(syms) != 3 {
		t.Fatalf("want 3 syms, got %d: %+v", len(syms), syms)
	}
	if syms[0].Kind != phpsyms.KindClass || syms[0].Name != "UserController" {
		t.Errorf("class sym wrong: %+v", syms[0])
	}
	if syms[0].Parent != "Controller" {
		t.Errorf("extends not captured: %+v", syms[0])
	}
	if len(syms[0].Implements) != 1 || syms[0].Implements[0] != "Authenticatable" {
		t.Errorf("implements not captured: %+v", syms[0])
	}
	if syms[1].Kind != phpsyms.KindMethod || syms[1].Name != "index" {
		t.Errorf("method[0] wrong: %+v", syms[1])
	}
	if syms[1].Parent != "UserController" {
		t.Errorf("method parent wrong: %+v", syms[1])
	}
	if syms[2].Name != "helper" || !hasMod(syms[2], "static") {
		t.Errorf("method[1] wrong: %+v", syms[2])
	}
}

func hasMod(s phpsyms.Symbol, want string) bool {
	for _, m := range s.Modifiers {
		if m == want {
			return true
		}
	}
	return false
}

func TestExtract_Interface(t *testing.T) {
	src := []byte(`<?php
interface Renderable extends Stringable, Arrayable
{
    public function render(): string;
}`)
	syms, _ := phpsyms.Extract("t.php", src)
	if len(syms) < 1 || syms[0].Kind != phpsyms.KindInterface {
		t.Fatalf("want interface symbol, got %+v", syms)
	}
	if syms[0].Name != "Renderable" {
		t.Errorf("interface name: %s", syms[0].Name)
	}
	if len(syms[0].Implements) != 2 || syms[0].Implements[0] != "Stringable" || syms[0].Implements[1] != "Arrayable" {
		t.Errorf("interface extends list (stored in Implements): %+v", syms[0].Implements)
	}
	// Method on interface should also be emitted (per Task 4's currentClass propagation).
	var sawRender bool
	for _, s := range syms {
		if s.Kind == phpsyms.KindMethod && s.Name == "render" {
			sawRender = true
			if s.Parent != "Renderable" {
				t.Errorf("render method parent: %s", s.Parent)
			}
		}
	}
	if !sawRender {
		t.Errorf("interface method 'render' not emitted; syms=%+v", syms)
	}
}

func TestExtract_Trait(t *testing.T) {
	src := []byte(`<?php
trait HasTimestamps
{
    public function touch(): void {}
}`)
	syms, _ := phpsyms.Extract("t.php", src)
	if len(syms) < 1 || syms[0].Kind != phpsyms.KindTrait {
		t.Fatalf("want trait symbol, got %+v", syms)
	}
	if syms[0].Name != "HasTimestamps" {
		t.Errorf("trait name: %s", syms[0].Name)
	}
	// touch method should be emitted with Parent=HasTimestamps.
	var sawTouch bool
	for _, s := range syms {
		if s.Kind == phpsyms.KindMethod && s.Name == "touch" {
			sawTouch = true
			if s.Parent != "HasTimestamps" {
				t.Errorf("touch method parent: %s", s.Parent)
			}
		}
	}
	if !sawTouch {
		t.Errorf("trait method 'touch' not emitted; syms=%+v", syms)
	}
}

func TestExtract_Function(t *testing.T) {
	src := []byte(`<?php
function helper(int $x): string { return ""; }
`)
	syms, _ := phpsyms.Extract("t.php", src)
	if len(syms) != 1 || syms[0].Kind != phpsyms.KindFunction || syms[0].Name != "helper" {
		t.Fatalf("want function helper, got %+v", syms)
	}
}

func TestExtract_FunctionAfterClass(t *testing.T) {
	// Verifies the brace-depth fix: a top-level function after a closing
	// class brace must be emitted as KindFunction (currentClass cleared),
	// NOT as KindMethod attached to the previous class.
	src := []byte(`<?php
class A {
    public function foo() {}
}

function topLevel() {}
`)
	syms, _ := phpsyms.Extract("t.php", src)
	var sawTopLevelFn bool
	var topLevelKind phpsyms.SymbolKind
	for _, s := range syms {
		if s.Name == "topLevel" {
			sawTopLevelFn = true
			topLevelKind = s.Kind
		}
	}
	if !sawTopLevelFn {
		t.Fatalf("topLevel not emitted; syms=%+v", syms)
	}
	if topLevelKind != phpsyms.KindFunction {
		t.Errorf("topLevel kind: got %v, want KindFunction", topLevelKind)
	}
}

func TestExtract_UseImports(t *testing.T) {
	src := []byte(`<?php
use App\Models\User;
use App\Http\{Request, Response};
use App\Auth\Guard as AuthGuard;
`)
	syms, _ := phpsyms.Extract("t.php", src)
	var names []string
	for _, s := range syms {
		if s.Kind == phpsyms.KindUseImport {
			names = append(names, s.Qualified)
		}
	}
	want := []string{
		"App\\Models\\User",
		"App\\Http\\Request",
		"App\\Http\\Response",
		"App\\Auth\\Guard",
	}
	if !equalUnordered(names, want) {
		t.Fatalf("use imports: got %v, want %v", names, want)
	}
}

func equalUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]int{}
	for _, x := range a {
		m[x]++
	}
	for _, x := range b {
		m[x]--
	}
	for _, v := range m {
		if v != 0 {
			return false
		}
	}
	return true
}

func TestExtract_Calls(t *testing.T) {
	src := []byte(`<?php
class Service {
    public function run() {
        Cache::get('key');
        $this->log('msg');
        helper(1, 2);
    }
}`)
	syms, _ := phpsyms.Extract("t.php", src)
	var static, method, function []string
	for _, s := range syms {
		switch s.Kind {
		case phpsyms.KindStaticCall:
			static = append(static, s.Parent+"::"+s.Name)
		case phpsyms.KindMethodCall:
			method = append(method, s.Receiver+"->"+s.Name)
		case phpsyms.KindFunctionCall:
			function = append(function, s.Name)
		}
	}
	if !sliceContains(static, "Cache::get") {
		t.Errorf("missing Cache::get; got %v", static)
	}
	if !sliceContains(method, "this->log") {
		t.Errorf("missing this->log; got %v", method)
	}
	if !sliceContains(function, "helper") {
		t.Errorf("missing helper; got %v", function)
	}
}

func sliceContains(s []string, want string) bool {
	for _, x := range s {
		if x == want {
			return true
		}
	}
	return false
}

func TestExtract_NamespacedStaticCall(t *testing.T) {
	src := []byte(`<?php
function run() {
    \App\Services\Cache::get('key');
}`)
	syms, _ := phpsyms.Extract("t.php", src)
	var found bool
	for _, s := range syms {
		if s.Kind == phpsyms.KindStaticCall && s.Name == "get" && s.Parent == "\\App\\Services\\Cache" {
			found = true
		}
	}
	if !found {
		t.Fatalf("namespaced static call not extracted; syms=%+v", syms)
	}
}

func TestExtract_FunctionCallNotConfusedWithDecl(t *testing.T) {
	src := []byte(`<?php
function realFunc() {
    other(1);
}`)
	syms, _ := phpsyms.Extract("t.php", src)
	// Expect: 1 KindFunction (realFunc) + 1 KindFunctionCall (other). NOT 2 of either.
	var fnCount, callCount int
	for _, s := range syms {
		if s.Kind == phpsyms.KindFunction {
			fnCount++
		}
		if s.Kind == phpsyms.KindFunctionCall {
			callCount++
		}
	}
	if fnCount != 1 {
		t.Errorf("want 1 KindFunction, got %d; syms=%+v", fnCount, syms)
	}
	if callCount != 1 {
		t.Errorf("want 1 KindFunctionCall, got %d; syms=%+v", callCount, syms)
	}
}
