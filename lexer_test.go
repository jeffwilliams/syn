package syn

import (
	"log"
	"os"
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

	lex, err := NewLexerFromXML([]rune(prog), "c.xml")
	assert.Nil(err)
	assert.NotNil(lex)
	if err != nil {
		t.FailNow()
	}

	DebugLogger = log.New(os.Stdout, "", 0)

	//t.Logf("Lexer is: %#v\n", lex)

	tokens := lex.Lex()

	t.Logf("Tokens returned were:\n")
	for _, tok := range tokens {
		t.Logf("  %s\n", tok)
	}

	assert.Equal([]Token{
		{Typ: Line, Value: []rune("#include")},
	},
		tokens)

}
