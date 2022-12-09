package syn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLexer(t *testing.T) {
	prog := `
#include <stdio.h>

int return_5() {
	return 5;
}

int main() {
	printf("value: %d\n", return_5());
}

`

	/*expectedToks := []Token{
		{Typ: Line, Value: []rune("#include")},
	}*/

	assert := assert.New(t)

	input := []rune(prog)
	lex, err := NewLexerFromXML(input, "c.xml")
	assert.Nil(err)
	assert.NotNil(lex)
	if err != nil {
		t.FailNow()
	}

	//DebugLogger = log.New(os.Stdout, "", 0)

	//t.Logf("Lexer is: %#v\n", lex)

	tokens := lex.Lex()

	t.Logf("Tokens returned were:\n")
	for _, tok := range tokens {
		t.Logf("  %s\n", tok)
	}

	// Make sure tokens are consecutive (in terms of rune index) and
	// the value matches the referenced indices
	for i, tok := range tokens {
		if tok.Typ == EOFType {
			continue
		}

		assert.Equal(tok.Value, input[tok.Start:tok.End],
			"token='%s' ref='%s' rawToken=(%+v)", string(tok.Value), string(input[tok.Start:tok.End]), tok)

		if i == 0 {
			continue
		}

		ptok := tokens[i-1]
		assert.Equal(tok.Start, ptok.End)
	}

	assert.Equal([]Token{
		{Typ: Text, Value: []rune("\n"), Start: 0, End: 1},
		{Typ: Line, Value: []rune("#include")},
	},
		tokens)

}
