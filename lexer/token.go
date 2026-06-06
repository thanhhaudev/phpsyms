package lexer

// TokenKind enumerates token classes emitted by the lexer.
type TokenKind int

const (
	TokError TokenKind = iota
	TokEOF
	TokInlineHTML // text outside <?php ... ?>
	TokOpenTag    // <?php
	TokCloseTag   // ?>
	TokIdent      // identifier (variable name without leading $)
	TokVariable   // $var
	TokKeyword    // class, function, public, etc. (use Value to disambiguate)
	TokString     // any string literal (single/double/heredoc/nowdoc), Value stripped of quotes
	TokNumber
	TokComment     // // ... or # ... or /* ... */
	TokLBrace      // {
	TokRBrace      // }
	TokLParen      // (
	TokRParen      // )
	TokLBracket    // [
	TokRBracket    // ]
	TokSemi        // ;
	TokComma       // ,
	TokDoubleColon // ::
	TokArrow       // ->
	TokBackslash   // \
	TokColon       // :
	TokQuestion    // ?
	TokEquals      // =
	TokOther       // any unhandled single-char punct
)

// Token is a single lexer output unit.
type Token struct {
	Kind      TokenKind
	Value     string // raw text for keyword/ident/string; unused for punct
	StartLine int    // 1-based line at first byte
	StartCol  int    // 1-based byte column at first byte
	EndLine   int    // 1-based line at byte AFTER last byte (exclusive)
	EndCol    int    // 1-based byte column at byte AFTER last byte (exclusive)
	StartByte int    // 0-based byte offset, inclusive
	EndByte   int    // 0-based byte offset, exclusive
}
