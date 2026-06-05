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

func TestLex_BlockCommentConsumesFinalByte(t *testing.T) {
	src := []byte("<?php /* abcX")
	toks := Lex("t.php", src)
	// Expect: OpenTag, Comment("/* abcX"), EOF — no leaked TokIdent for 'X'.
	var sawComment bool
	for _, tk := range toks {
		if tk.Kind == TokComment {
			sawComment = true
			if tk.Value != "/* abcX" {
				t.Errorf("unterminated comment body: got %q want %q", tk.Value, "/* abcX")
			}
		}
		if tk.Kind == TokIdent && tk.Value == "X" {
			t.Errorf("byte 'X' leaked out of unterminated comment: %+v", toks)
		}
	}
	if !sawComment {
		t.Fatal("no comment token emitted")
	}
}

func TestLex_SingleStringValueStrippedOfQuotes(t *testing.T) {
	src := []byte("<?php $x = 'hello';")
	toks := Lex("t.php", src)
	var found bool
	for _, tk := range toks {
		if tk.Kind == TokString {
			found = true
			if tk.Value != "hello" {
				t.Errorf("single-quoted Value: got %q want %q", tk.Value, "hello")
			}
		}
	}
	if !found {
		t.Fatal("no string token emitted")
	}
}

func TestLex_TokenStartPositionAtFirstByte(t *testing.T) {
	// Verify multi-byte tokens record StartLine/StartCol at the FIRST byte,
	// not after the last byte. Source: `<?php class Foo`
	// Positions (1-based col): `<` at col 1, `c` of `class` at col 7, `F` of `Foo` at col 13.
	src := []byte("<?php class Foo")
	toks := Lex("t.php", src)
	if len(toks) < 3 {
		t.Fatalf("need at least 3 tokens, got %d", len(toks))
	}
	// toks[0] = TokOpenTag "<?php", col 1
	if toks[0].StartCol != 1 || toks[0].StartLine != 1 {
		t.Errorf("OpenTag startpos: %+v", toks[0])
	}
	// toks[1] = TokKeyword "class", col 7 (after "<?php ")
	if toks[1].Value != "class" {
		t.Fatalf("unexpected toks[1]: %+v", toks[1])
	}
	if toks[1].StartCol != 7 || toks[1].StartLine != 1 {
		t.Errorf("keyword 'class' startpos: got col=%d line=%d, want col=7 line=1; tok=%+v", toks[1].StartCol, toks[1].StartLine, toks[1])
	}
	// toks[2] = TokIdent "Foo", col 13
	if toks[2].Value != "Foo" {
		t.Fatalf("unexpected toks[2]: %+v", toks[2])
	}
	if toks[2].StartCol != 13 {
		t.Errorf("ident 'Foo' startpos: got col=%d, want col=13; tok=%+v", toks[2].StartCol, toks[2])
	}
}
