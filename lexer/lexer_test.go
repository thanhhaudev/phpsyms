package lexer

import "testing"

func TestLex_EmptyFile(t *testing.T) {
	toks := Lex("", []byte(""))
	if len(toks) != 1 || toks[0].Kind != TokEOF {
		t.Fatalf("want single EOF, got %+v", toks)
	}
}

func TestLex_PlainPHPClass(t *testing.T) {
	src := []byte("<?php class Foo {}")
	toks := Lex("test.php", src)
	wantKinds := []TokenKind{TokOpenTag, TokKeyword, TokIdent, TokLBrace, TokRBrace, TokEOF}
	if len(toks) != len(wantKinds) {
		t.Fatalf("token count: got %d, want %d. tokens=%+v", len(toks), len(wantKinds), toks)
	}
	for i, k := range wantKinds {
		if toks[i].Kind != k {
			t.Errorf("toks[%d].Kind = %v, want %v", i, toks[i].Kind, k)
		}
	}
	if toks[1].Value != "class" {
		t.Errorf("keyword value: got %q, want %q", toks[1].Value, "class")
	}
	if toks[2].Value != "Foo" {
		t.Errorf("ident value: got %q, want %q", toks[2].Value, "Foo")
	}
}

func TestLex_DoubleStringWithVarInterpolation(t *testing.T) {
	src := []byte(`<?php $x = "hello $name end";`)
	toks := Lex("t.php", src)
	for _, tk := range toks {
		if tk.Kind == TokVariable && tk.Value == "$name" {
			t.Fatalf("$name leaked out of string: %+v", toks)
		}
	}
}

func TestLex_Heredoc(t *testing.T) {
	src := []byte("<?php $x = <<<EOT\nhello $name\nEOT;\n")
	toks := Lex("t.php", src)
	for _, tk := range toks {
		if tk.Kind == TokVariable && tk.Value == "$name" {
			t.Fatalf("$name leaked from heredoc: %+v", toks)
		}
	}
}

func TestLex_LineComments(t *testing.T) {
	src := []byte("<?php // class Foo\n# class Bar\nclass Baz {}")
	toks := Lex("t.php", src)
	var idents []string
	for _, tk := range toks {
		if tk.Kind == TokIdent {
			idents = append(idents, tk.Value)
		}
	}
	if len(idents) != 1 || idents[0] != "Baz" {
		t.Fatalf("expected only Baz; got idents=%v", idents)
	}
}

func TestLex_BlockComment(t *testing.T) {
	src := []byte("<?php /* class Foo */ class Bar {}")
	toks := Lex("t.php", src)
	var idents []string
	for _, tk := range toks {
		if tk.Kind == TokIdent {
			idents = append(idents, tk.Value)
		}
	}
	if len(idents) != 1 || idents[0] != "Bar" {
		t.Fatalf("expected only Bar; got idents=%v", idents)
	}
}

func TestLex_NeverPanicsOnGarbage(t *testing.T) {
	inputs := [][]byte{
		[]byte("<?php $"),
		[]byte("<?php \"unterminated"),
		[]byte("<?php <<<EOT\nno end"),
		[]byte("<?php /* unterminated comment"),
		[]byte("\xff\xfe\x00<?php class"),
	}
	for i, in := range inputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("input %d panicked: %v", i, r)
				}
			}()
			_ = Lex("garbage.php", in)
		}()
	}
}
