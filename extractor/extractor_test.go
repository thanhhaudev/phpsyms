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
