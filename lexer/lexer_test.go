package lexer

import (
	"strings"
	"testing"
)

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

func TestLex_Nowdoc(t *testing.T) {
	src := []byte("<?php $x = <<<'EOT'\nhello $name\nEOT;\n")
	toks := Lex("t.php", src)
	for _, tk := range toks {
		if tk.Kind == TokVariable && tk.Value == "$name" {
			t.Fatalf("$name leaked from nowdoc: %+v", toks)
		}
	}
	// Positive assertion: the string body must contain the literal "$name" text.
	var sawString bool
	for _, tk := range toks {
		if tk.Kind == TokString {
			sawString = true
			if !strings.Contains(tk.Value, "$name") {
				t.Errorf("nowdoc body lost $name text: %q", tk.Value)
			}
		}
	}
	if !sawString {
		t.Fatal("no TokString emitted for nowdoc")
	}
}

func TestLex_HeredocIndented(t *testing.T) {
	// PHP 7.3+ allows the closing label to be indented; the indentation is
	// stripped from each line of the body when PHP interprets it. The lexer
	// only needs to recognize the closing label and exit heredoc mode.
	src := []byte("<?php $x = <<<EOT\n        indented content\n        EOT;\n")
	toks := Lex("t.php", src)
	// The semicolon after the indented EOT must be emitted as TokSemi —
	// otherwise the closing label wasn't recognized.
	var sawSemi bool
	for _, tk := range toks {
		if tk.Kind == TokSemi {
			sawSemi = true
		}
	}
	if !sawSemi {
		t.Fatalf("indented heredoc closing label not recognized; tokens=%+v", toks)
	}
}

func TestLex_BracedInterpolation(t *testing.T) {
	src := []byte(`<?php $x = "{$obj->prop} done";`)
	toks := Lex("t.php", src)
	// $obj must NOT leak as a TokVariable.
	for _, tk := range toks {
		if tk.Kind == TokVariable && tk.Value == "$obj" {
			t.Fatalf("$obj leaked from braced interp: %+v", toks)
		}
	}
	// A TokString must be present.
	var sawString bool
	for _, tk := range toks {
		if tk.Kind == TokString {
			sawString = true
		}
	}
	if !sawString {
		t.Fatal("no string token emitted")
	}
}

func TestLex_NullCoalesceReturnType(t *testing.T) {
	// Verify the lexer doesn't choke on PHP 7.1+ nullable type + null-coalesce
	// in a single function signature + body.
	src := []byte("<?php function f(?array $a): ?string { return $a['k'] ?? null; }")
	toks := Lex("t.php", src)
	var sawF bool
	for _, tk := range toks {
		if tk.Kind == TokIdent && tk.Value == "f" {
			sawF = true
		}
	}
	if !sawF {
		t.Fatalf("function name not tokenized; tokens=%+v", toks)
	}
}

func TestLex_VariableEdgeCases(t *testing.T) {
	cases := []struct {
		src      string
		wantVars []string
	}{
		{`<?php $a = 1;`, []string{"$a"}},
		{`<?php $_foo = 1;`, []string{"$_foo"}},
		{`<?php $snake_case = 1;`, []string{"$snake_case"}},
		{`<?php $camelCase = 1;`, []string{"$camelCase"}},
		{`<?php $with123digits = 1;`, []string{"$with123digits"}},
		{`<?php $a + $b;`, []string{"$a", "$b"}},
		{`<?php // $a comment`, nil},
		{`<?php "string $a"`, nil},
		{`<?php 'literal $a'`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			toks := Lex("t.php", []byte(tc.src))
			var got []string
			for _, tk := range toks {
				if tk.Kind == TokVariable {
					got = append(got, tk.Value)
				}
			}
			if !stringSliceEqual(got, tc.wantVars) {
				t.Errorf("vars: got %v, want %v", got, tc.wantVars)
			}
		})
	}
}

// Bare $ with no following ident must not emit TokVariable and must not panic.
func TestLex_BareDollarSign(t *testing.T) {
	src := []byte("<?php $ + 1;")
	toks := Lex("t.php", src)
	for _, tk := range toks {
		if tk.Kind == TokVariable {
			t.Fatalf("bare $ should not emit TokVariable, got %+v", toks)
		}
	}
}

func FuzzLex(f *testing.F) {
	f.Add([]byte("<?php class Foo {}"))
	f.Add([]byte("<?php $x = \"hi $name\";"))
	f.Add([]byte("<?php /* unterminated"))
	f.Add([]byte("<?php <<<EOT\nhello\nEOT;\n"))
	f.Add([]byte("<?php <<<'EOT'\nhello\nEOT;\n"))
	f.Fuzz(func(t *testing.T, src []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic on input %q: %v", src, r)
			}
		}()
		toks := Lex("fuzz.php", src)
		if len(toks) == 0 || toks[len(toks)-1].Kind != TokEOF {
			t.Fatalf("Lex did not terminate with TokEOF; tokens=%v", toks)
		}
	})
}

func TestLex_TokenEndPosition(t *testing.T) {
	// For `<?php class Foo`:
	// - `<?php` spans byte 0..5, EndCol should be 6 (exclusive col after last byte)
	// - `class` (after the space at byte 5) spans byte 6..11, EndCol should be 12
	// - `Foo` spans byte 12..15, EndCol should be 16
	src := []byte("<?php class Foo")
	toks := Lex("t.php", src)
	if len(toks) < 3 {
		t.Fatalf("need at least 3 tokens, got %d", len(toks))
	}
	// toks[0] = TokOpenTag "<?php"
	if toks[0].EndCol != 6 || toks[0].EndByte != 5 {
		t.Errorf("OpenTag end: got col=%d byte=%d, want col=6 byte=5; tok=%+v", toks[0].EndCol, toks[0].EndByte, toks[0])
	}
	// toks[1] = TokKeyword "class" starting col 7
	if toks[1].Value != "class" {
		t.Fatalf("unexpected toks[1]: %+v", toks[1])
	}
	if toks[1].EndCol != 12 {
		t.Errorf("class EndCol: got %d, want 12; tok=%+v", toks[1].EndCol, toks[1])
	}
	// toks[2] = TokIdent "Foo" starting col 13
	if toks[2].Value != "Foo" {
		t.Fatalf("unexpected toks[2]: %+v", toks[2])
	}
	if toks[2].EndCol != 16 {
		t.Errorf("Foo EndCol: got %d, want 16; tok=%+v", toks[2].EndCol, toks[2])
	}
}

func TestLex_PunctEndPosition(t *testing.T) {
	// Verify single-byte punct has EndCol = StartCol + 1.
	src := []byte("<?php {")
	toks := Lex("t.php", src)
	var lbrace *Token
	for i := range toks {
		if toks[i].Kind == TokLBrace {
			lbrace = &toks[i]
			break
		}
	}
	if lbrace == nil {
		t.Fatal("no LBrace token")
	}
	if lbrace.EndCol != lbrace.StartCol+1 {
		t.Errorf("LBrace EndCol: got %d, StartCol=%d, want EndCol=StartCol+1", lbrace.EndCol, lbrace.StartCol)
	}
}

func TestLex_TwoBytePunctEndPosition(t *testing.T) {
	src := []byte("<?php ::->")
	toks := Lex("t.php", src)
	var doublecolon, arrow *Token
	for i := range toks {
		if toks[i].Kind == TokDoubleColon {
			doublecolon = &toks[i]
		}
		if toks[i].Kind == TokArrow {
			arrow = &toks[i]
		}
	}
	if doublecolon == nil || doublecolon.EndCol != doublecolon.StartCol+2 {
		t.Errorf("DoubleColon end-pos wrong: %+v", doublecolon)
	}
	if arrow == nil || arrow.EndCol != arrow.StartCol+2 {
		t.Errorf("Arrow end-pos wrong: %+v", arrow)
	}
}

// Helper — only add if not already present in the test file.
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
