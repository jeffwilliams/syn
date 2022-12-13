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

	assert := assert.New(t)

	input := []rune(prog)
	lex, err := NewLexerFromXML(input, "test_data/c.xml")
	assert.Nil(err)
	assert.NotNil(lex)
	if err != nil {
		t.FailNow()
	}

	//DebugLogger = log.New(os.Stdout, "", 0)

	var tokens []Token
	it := Coalesce(lex)
	for {
		tok, err := it.Next()
		assert.Nil(err)
		if err != nil {
			t.FailNow()
		}

		if tok.Typ == Error || tok.Typ == EOFType {
			break
		}
		tokens = append(tokens, tok)
	}

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

	expected := []Token{
		{Typ: Text, Value: []rune("\n"), Start: 0, End: 1},
		{Typ: CommentPreproc, Value: []rune("#include"), Start: 1, End: 9},
		{Typ: Text, Value: []rune(" "), Start: 9, End: 10},
		{Typ: CommentPreprocFile, Value: []rune("<stdio.h>"), Start: 10, End: 19},
		{Typ: CommentPreproc, Value: []rune("\n"), Start: 19, End: 20},
		{Typ: Text, Value: []rune("\n"), Start: 20, End: 21},
		{Typ: KeywordType, Value: []rune("int"), Start: 21, End: 24},
		{Typ: Text, Value: []rune(" "), Start: 24, End: 25},
		{Typ: NameFunction, Value: []rune("return_5"), Start: 25, End: 33},
		{Typ: Punctuation, Value: []rune("()"), Start: 33, End: 35},
		{Typ: Text, Value: []rune(" "), Start: 35, End: 36},
		{Typ: Punctuation, Value: []rune("{"), Start: 36, End: 37},
		{Typ: Text, Value: []rune("\n\t"), Start: 37, End: 39},
		{Typ: Keyword, Value: []rune("return"), Start: 39, End: 45},
		{Typ: Text, Value: []rune(" "), Start: 45, End: 46},
		{Typ: LiteralNumberInteger, Value: []rune("5"), Start: 46, End: 47},
		{Typ: Punctuation, Value: []rune(";"), Start: 47, End: 48},
		{Typ: Text, Value: []rune("\n"), Start: 48, End: 49},
		{Typ: Punctuation, Value: []rune("}"), Start: 49, End: 50},
		{Typ: Text, Value: []rune("\n\n"), Start: 50, End: 52},
		{Typ: KeywordType, Value: []rune("int"), Start: 52, End: 55},
		{Typ: Text, Value: []rune(" "), Start: 55, End: 56},
		{Typ: NameFunction, Value: []rune("main"), Start: 56, End: 60},
		{Typ: Punctuation, Value: []rune("()"), Start: 60, End: 62},
		{Typ: Text, Value: []rune(" "), Start: 62, End: 63},
		{Typ: Punctuation, Value: []rune("{"), Start: 63, End: 64},
		{Typ: Text, Value: []rune("\n\t"), Start: 64, End: 66},
		{Typ: NameFunction, Value: []rune("printf"), Start: 66, End: 72},
		{Typ: Punctuation, Value: []rune("("), Start: 72, End: 73},
		{Typ: LiteralStringAffix, Value: []rune(""), Start: 73, End: 73},
		{Typ: LiteralString, Value: []rune(`"value: %d`), Start: 73, End: 83},
		{Typ: LiteralStringEscape, Value: []rune(`\n`), Start: 83, End: 85},
		{Typ: LiteralString, Value: []rune(`"`), Start: 85, End: 86},
		{Typ: Punctuation, Value: []rune(","), Start: 86, End: 87},
		{Typ: Text, Value: []rune(" "), Start: 87, End: 88},
		{Typ: NameFunction, Value: []rune("return_5"), Start: 88, End: 96},
		{Typ: Punctuation, Value: []rune("());"), Start: 96, End: 100},
		{Typ: Text, Value: []rune("\n"), Start: 100, End: 101},
		{Typ: Punctuation, Value: []rune("}"), Start: 101, End: 102},
		{Typ: Text, Value: []rune("\n\n"), Start: 102, End: 104},
	}

	assert.Equal(expected, tokens)

	for i, tok := range expected {
		if i >= len(tokens) {
			break
		}
		if !tokensEqual(&tok, &tokens[i]) {
			t.Fatalf("Token %d doesn't match. Expected (%s) but got (%s)", i, tok, tokens[i])
		}
	}

}

func tokensEqual(t1, t2 *Token) bool {
	return t1.Typ == t2.Typ && string(t1.Value) == string(t2.Value) && t1.Start == t2.Start && t1.End == t2.End
}
