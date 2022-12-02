package syn

import (
	"github.com/stretchr/testify/assert"
	"testing"
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
	assert := assert.New(t)

	lex, err := NewLexerFromXML([]byte(prog), "c.xml")
	assert.Nil(err)
	assert.NotNil(lex)

	tokens := lex.Lex()

	assert.Equal([]Token{
		{Typ: Line, Value: []byte("#include")},
	},
		tokens)
}
