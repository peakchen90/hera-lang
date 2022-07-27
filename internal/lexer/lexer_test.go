package lexer

import (
	"testing"
)

var tokenFixtures = [...]string{
	`"abc"`,
	`"\n\t\v\?\""`,
	`"\1a"`,
	`"\7777"`,
	`"\8"`,
	`"\xff"`,
	`"\x00"`,
	`"""a
b""\"
"""`,
	`0`,
	`1.2`,
	`-12`,
	`-12.34`,
}

func validTokenMap() {
	for i := TTEof; i < TTUnref; i++ {
		if tokenMetaTable[i].Type != i {
			panic("")
		}
	}
}

func TestLexer(t *testing.T) {
	validTokenMap()

	for _, fixture := range tokenFixtures {
		lexer := NewLexer([]rune(fixture))
		lexer.Next()
	}

	for _, fixture := range Keywords {
		lexer := NewLexer([]rune(fixture))
		lexer.Next()
	}
}
