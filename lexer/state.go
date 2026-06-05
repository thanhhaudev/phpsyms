package lexer

// state identifies the current lexer mode.
type state int

const (
	stateText         state = iota // outside <?php
	statePHP                       // inside <?php ... ?>
	stateDoubleString              // inside "..." (interpolation possible)
	stateHeredoc                   // inside <<<LABEL ... LABEL
	stateLineComment               // // ... or # ...
	stateBlockComment              // /* ... */
)
