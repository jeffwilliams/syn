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
	lex, err := NewLexerFromXMLFile("test_data/c.xml")
	assert.Nil(err)
	assert.NotNil(lex)
	if err != nil {
		t.FailNow()
	}

	//DebugLogger = log.New(os.Stdout, "", 0)

	tokens, err := tokenize(Coalesce(lex.Tokenise(input)))
	if err != nil {
		t.Fatalf("Tokenizing returned error: %v\n", err)
	}

	dumpTokens(t, tokens)

	// Make sure tokens are consecutive (in terms of rune index) and
	// the value matches the referenced indices
	for i, tok := range tokens {
		if tok.Type == EOFType {
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
		{Type: Text, Value: []rune("\n"), Start: 0, End: 1},
		{Type: CommentPreproc, Value: []rune("#include"), Start: 1, End: 9},
		{Type: Text, Value: []rune(" "), Start: 9, End: 10},
		{Type: CommentPreprocFile, Value: []rune("<stdio.h>"), Start: 10, End: 19},
		{Type: CommentPreproc, Value: []rune("\n"), Start: 19, End: 20},
		{Type: Text, Value: []rune("\n"), Start: 20, End: 21},
		{Type: KeywordType, Value: []rune("int"), Start: 21, End: 24},
		{Type: Text, Value: []rune(" "), Start: 24, End: 25},
		{Type: NameFunction, Value: []rune("return_5"), Start: 25, End: 33},
		{Type: Punctuation, Value: []rune("()"), Start: 33, End: 35},
		{Type: Text, Value: []rune(" "), Start: 35, End: 36},
		{Type: Punctuation, Value: []rune("{"), Start: 36, End: 37},
		{Type: Text, Value: []rune("\n\t"), Start: 37, End: 39},
		{Type: Keyword, Value: []rune("return"), Start: 39, End: 45},
		{Type: Text, Value: []rune(" "), Start: 45, End: 46},
		{Type: LiteralNumberInteger, Value: []rune("5"), Start: 46, End: 47},
		{Type: Punctuation, Value: []rune(";"), Start: 47, End: 48},
		{Type: Text, Value: []rune("\n"), Start: 48, End: 49},
		{Type: Punctuation, Value: []rune("}"), Start: 49, End: 50},
		{Type: Text, Value: []rune("\n\n"), Start: 50, End: 52},
		{Type: KeywordType, Value: []rune("int"), Start: 52, End: 55},
		{Type: Text, Value: []rune(" "), Start: 55, End: 56},
		{Type: NameFunction, Value: []rune("main"), Start: 56, End: 60},
		{Type: Punctuation, Value: []rune("()"), Start: 60, End: 62},
		{Type: Text, Value: []rune(" "), Start: 62, End: 63},
		{Type: Punctuation, Value: []rune("{"), Start: 63, End: 64},
		{Type: Text, Value: []rune("\n\t"), Start: 64, End: 66},
		{Type: NameFunction, Value: []rune("printf"), Start: 66, End: 72},
		{Type: Punctuation, Value: []rune("("), Start: 72, End: 73},
		{Type: LiteralStringAffix, Value: []rune(""), Start: 73, End: 73},
		{Type: LiteralString, Value: []rune(`"value: %d`), Start: 73, End: 83},
		{Type: LiteralStringEscape, Value: []rune(`\n`), Start: 83, End: 85},
		{Type: LiteralString, Value: []rune(`"`), Start: 85, End: 86},
		{Type: Punctuation, Value: []rune(","), Start: 86, End: 87},
		{Type: Text, Value: []rune(" "), Start: 87, End: 88},
		{Type: NameFunction, Value: []rune("return_5"), Start: 88, End: 96},
		{Type: Punctuation, Value: []rune("());"), Start: 96, End: 100},
		{Type: Text, Value: []rune("\n"), Start: 100, End: 101},
		{Type: Punctuation, Value: []rune("}"), Start: 101, End: 102},
		{Type: Text, Value: []rune("\n\n"), Start: 102, End: 104},
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

/*
func TestLexerNoCoalesing(t *testing.T) {
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

	tokens, err := tokenize(lex)
	if err != nil {
		t.Fatalf("Tokenizing returned error: %v\n", err)
	}

	dumpTokens(t, tokens)

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
*/

func tokenize(it Iterator) (tokens []Token, err error) {
	//it := Coalesce(lex)
	for {
		var tok Token
		tok, err = it.Next()
		if err != nil {
			return
		}

		if tok.Type == Error || tok.Type == EOFType {
			break
		}
		tokens = append(tokens, tok)
	}
	return
}

func tokenizeAtMost(it Iterator, n int) (tokens []Token, err error) {
	//it := Coalesce(lex)
	for n > 0 {
		var tok Token
		tok, err = it.Next()
		if err != nil {
			return
		}

		if tok.Type == Error || tok.Type == EOFType {
			break
		}
		tokens = append(tokens, tok)
		n--
	}
	return
}

func dumpTokens(t *testing.T, tokens []Token) {
	t.Logf("Tokens returned were:\n")
	for _, tok := range tokens {
		t.Logf("  %s\n", tok)
	}
}

func tokensEqual(t1, t2 *Token) bool {
	return t1.Type == t2.Type && string(t1.Value) == string(t2.Value) && t1.Start == t2.Start && t1.End == t2.End
}

func TestLexerStateRestoring(t *testing.T) {
	prog := `
#include <stdio.h>

int return_5() {
	return 5;
}

int main() {
	printf("value: %d\n", return_5());
}

`
	expected := []Token{
		{Type: Text, Value: []rune("\n"), Start: 0, End: 1},
		{Type: CommentPreproc, Value: []rune("#include"), Start: 1, End: 9},
		{Type: Text, Value: []rune(" "), Start: 9, End: 10},
		{Type: CommentPreprocFile, Value: []rune("<stdio.h>"), Start: 10, End: 19},
		{Type: CommentPreproc, Value: []rune("\n"), Start: 19, End: 20},
		{Type: Text, Value: []rune("\n"), Start: 20, End: 21},
		{Type: KeywordType, Value: []rune("int"), Start: 21, End: 24},
		{Type: Text, Value: []rune(" "), Start: 24, End: 25},
		{Type: NameFunction, Value: []rune("return_5"), Start: 25, End: 33},
		{Type: Punctuation, Value: []rune("()"), Start: 33, End: 35},
		{Type: Text, Value: []rune(" "), Start: 35, End: 36},
		{Type: Punctuation, Value: []rune("{"), Start: 36, End: 37},
		{Type: Text, Value: []rune("\n\t"), Start: 37, End: 39},
		{Type: Keyword, Value: []rune("return"), Start: 39, End: 45},
		{Type: Text, Value: []rune(" "), Start: 45, End: 46},
		{Type: LiteralNumberInteger, Value: []rune("5"), Start: 46, End: 47},
		{Type: Punctuation, Value: []rune(";"), Start: 47, End: 48},
		{Type: Text, Value: []rune("\n"), Start: 48, End: 49},
		{Type: Punctuation, Value: []rune("}"), Start: 49, End: 50},
		{Type: Text, Value: []rune("\n\n"), Start: 50, End: 52},
		{Type: KeywordType, Value: []rune("int"), Start: 52, End: 55},
		{Type: Text, Value: []rune(" "), Start: 55, End: 56},
		{Type: NameFunction, Value: []rune("main"), Start: 56, End: 60},
		{Type: Punctuation, Value: []rune("()"), Start: 60, End: 62},
		{Type: Text, Value: []rune(" "), Start: 62, End: 63},
		{Type: Punctuation, Value: []rune("{"), Start: 63, End: 64},
		{Type: Text, Value: []rune("\n\t"), Start: 64, End: 66},
		{Type: NameFunction, Value: []rune("printf"), Start: 66, End: 72},
		{Type: Punctuation, Value: []rune("("), Start: 72, End: 73},
		{Type: LiteralStringAffix, Value: []rune(""), Start: 73, End: 73},
		{Type: LiteralString, Value: []rune(`"value: %d`), Start: 73, End: 83},
		{Type: LiteralStringEscape, Value: []rune(`\n`), Start: 83, End: 85},
		{Type: LiteralString, Value: []rune(`"`), Start: 85, End: 86},
		{Type: Punctuation, Value: []rune(","), Start: 86, End: 87},
		{Type: Text, Value: []rune(" "), Start: 87, End: 88},
		{Type: NameFunction, Value: []rune("return_5"), Start: 88, End: 96},
		{Type: Punctuation, Value: []rune("());"), Start: 96, End: 100},
		{Type: Text, Value: []rune("\n"), Start: 100, End: 101},
		{Type: Punctuation, Value: []rune("}"), Start: 101, End: 102},
		{Type: Text, Value: []rune("\n\n"), Start: 102, End: 104},
	}

	assert := assert.New(t)

	makeLexer := func() *Lexer {
		lex, err := NewLexerFromXMLFile("test_data/c.xml")
		assert.Nil(err)
		assert.NotNil(lex)
		if err != nil {
			t.FailNow()
		}
		return lex
	}

	input := []rune(prog)

	// The idea here is to text that saving and restoring the state of a lexer works.
	// We lex some of the input, then save and restore the state, then continue lexing.
	// The result should be the same as if we never saved and restored the state.
	for i := 1; i < len(expected); i++ {
		t.Logf("Tokenizing %d tokens, saving/restoring state, then continuing\n", i)

		lex := makeLexer()
		lit := lex.Tokenise(input)
		it := Coalesce(lit)

		tokens, err := tokenizeAtMost(it, i)
		if err != nil {
			t.Fatalf("Tokenizing returned error: %v\n", err)
		}

		// Save state
		state := it.State()

		// Tokenize all the rest of the tokens
		_, err = tokenize(it)
		if err != nil {
			t.Fatalf("Tokenizing returned error: %v\n", err)
		}

		// Restore the state back to an earlier place
		it.SetState(state)

		// Now tokenize from that place
		moreTokens, err := tokenize(it)
		if err != nil {
			t.Fatalf("Tokenizing returned error: %v\n", err)
		}

		tokens = append(tokens, moreTokens...)

		dumpTokens(t, tokens)

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

}

func TestEnsureLF(t *testing.T) {
	text := "line1\r\nline2\r\nline3\r\n"

	assert := assert.New(t)

	runes, omap := ensureLF([]rune(text))
	stripped := string(runes)
	assert.Equal("line1\nline2\nline3\n", stripped)

	expectedTransitions := []int{5, 12, 19}
	assert.Equal(expectedTransitions, omap.transitions)

}

func TestLexerCRLF(t *testing.T) {
	prog := "\r\n#include <stdio.h>\r\n\r\nint return_5() {\r\n	return 5;\r\n}\r\n"

	assert := assert.New(t)

	input := []rune(prog)
	lex, err := NewLexerFromXMLFile("test_data/c.xml")
	assert.Nil(err)
	assert.NotNil(lex)
	if err != nil {
		t.FailNow()
	}

	//DebugLogger = log.New(os.Stdout, "", 0)

	tokens, err := tokenize(Coalesce(lex.Tokenise(input)))
	if err != nil {
		t.Fatalf("Tokenizing returned error: %v\n", err)
	}

	dumpTokens(t, tokens)

	// Make sure tokens are consecutive (in terms of rune index) and
	// the value matches the referenced indices
	for i, tok := range tokens {
		if tok.Type == EOFType {
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
		{Type: Text, Value: []rune("\r\n"), Start: 0, End: 2},
		{Type: CommentPreproc, Value: []rune("#include"), Start: 2, End: 10},
		{Type: Text, Value: []rune(" "), Start: 10, End: 11},
		{Type: CommentPreprocFile, Value: []rune("<stdio.h>"), Start: 11, End: 20},
		{Type: CommentPreproc, Value: []rune("\r\n"), Start: 20, End: 22},
		{Type: Text, Value: []rune("\r\n"), Start: 22, End: 24},
		{Type: KeywordType, Value: []rune("int"), Start: 24, End: 27},
		{Type: Text, Value: []rune(" "), Start: 27, End: 28},
		{Type: NameFunction, Value: []rune("return_5"), Start: 28, End: 36},
		{Type: Punctuation, Value: []rune("()"), Start: 36, End: 38},
		{Type: Text, Value: []rune(" "), Start: 38, End: 39},
		{Type: Punctuation, Value: []rune("{"), Start: 39, End: 40},
		{Type: Text, Value: []rune("\r\n\t"), Start: 40, End: 43},
		{Type: Keyword, Value: []rune("return"), Start: 43, End: 49},
		{Type: Text, Value: []rune(" "), Start: 49, End: 50},
		{Type: LiteralNumberInteger, Value: []rune("5"), Start: 50, End: 51},
		{Type: Punctuation, Value: []rune(";"), Start: 51, End: 52},
		{Type: Text, Value: []rune("\r\n"), Start: 52, End: 54},
		{Type: Punctuation, Value: []rune("}"), Start: 54, End: 55},
		{Type: Text, Value: []rune("\r\n"), Start: 55, End: 57},
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
