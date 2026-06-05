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
