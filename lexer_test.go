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
	lex, err := NewLexerFromXMLFile("lexers/embedded/c.xml")
	assert.Nil(err)
	assert.NotNil(lex)
	if err != nil {
		t.FailNow()
	}

	//DebugLogger = log.New(os.Stdout, "", 0)

	tokens, err := tokenize(lex.Tokenise(input))
	if err != nil {
		t.Fatalf("Tokenizing returned error: %v. Input text length was %d\n", err, len(input))
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

func tokenize(it Iterator) (tokens []Token, err error) {
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

func tokenizeAndLog(it Iterator, t *testing.T) (tokens []Token, err error) {
	for {
		var tok Token
		tok, err = it.Next()
		if err != nil {
			return
		}

		t.Logf("test tokenize: tok = %s\n", tok)

		if tok.Type == Error || tok.Type == EOFType {
			break
		}
		tokens = append(tokens, tok)
	}
	return
}

func tokenizeAtMost(it Iterator, n int) (tokens []Token, err error) {
	for n > 0 {
		var tok Token
		tok, err = it.Next()
		if err != nil {
			return
		}

		if tok.Type == EOFType {
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
		lex, err := NewLexerFromXMLFile("lexers/embedded/c.xml")
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
		it := lex.Tokenise(input)

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
		moreTokens, err := tokenizeAndLog(it, t)
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
	lex, err := NewLexerFromXMLFile("lexers/embedded/c.xml")
	assert.Nil(err)
	assert.NotNil(lex)
	if err != nil {
		t.FailNow()
	}

	//DebugLogger = log.New(os.Stdout, "", 0)

	tokens, err := tokenize(lex.Tokenise(input))
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

func TestMarkdown(t *testing.T) {
	doc := `# heading
text
`

	assert := assert.New(t)

	input := []rune(doc)
	lex, err := NewLexerFromXMLFile("lexers/embedded/markdown.xml")
	assert.Nil(err)
	assert.NotNil(lex)
	if err != nil {
		t.FailNow()
	}

	//DebugLogger = log.New(os.Stdout, "", 0)

	tokens, err := tokenize(lex.Tokenise(input))
	if err != nil {
		t.Fatalf("Tokenizing returned error: %v\n", err)
	}

	dumpTokens(t, tokens)

	expected := []Token{
		{Type: GenericHeading, Value: []rune("# heading\n"), Start: 0, End: 10},
		{Type: Other, Value: []rune("text\n"), Start: 10, End: 15},
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

func TestEqual(t *testing.T) {
	prog := `int var1;
int var2;
int var3;
	`

	assert := assert.New(t)

	input := []rune(prog)
	lex, err := NewLexerFromXMLFile("lexers/embedded/c.xml")
	assert.Nil(err)
	assert.NotNil(lex)
	if err != nil {
		t.FailNow()
	}

	it1 := lex.Tokenise(input)
	it2 := lex.Tokenise(input)

	// tokenize it1 until just after line 1
	expected := []Token{
		{Type: KeywordType, Value: []rune("int"), Start: 0, End: 3},
		{Type: Text, Value: []rune(" "), Start: 3, End: 4},
		{Type: Name, Value: []rune("var1"), Start: 4, End: 8},
		{Type: Punctuation, Value: []rune(";"), Start: 8, End: 9},
		{Type: Text, Value: []rune("\n"), Start: 9, End: 10},
	}

	tokens, err := tokenizeAtMost(it1, 5)
	assert.Nil(err)
	// make sure we're in the right place
	assert.Equal(expected, tokens)

	// tokenize it2 until just after line 1
	tokens, err = tokenizeAtMost(it2, 5)
	assert.Nil(err)
	// make sure we're in the right place
	assert.Equal(expected, tokens)

	// Make sure it1 state equals it2 state
	assert.True(it1.State().Equal(it2.State()))

	// Move it2 to the end of line 2
	expected = []Token{
		{Type: KeywordType, Value: []rune("int"), Start: 10, End: 13},
		{Type: Text, Value: []rune(" "), Start: 13, End: 14},
		{Type: Name, Value: []rune("var2"), Start: 14, End: 18},
		{Type: Punctuation, Value: []rune(";"), Start: 18, End: 19},
		{Type: Text, Value: []rune("\n"), Start: 19, End: 20},
	}

	tokens, err = tokenizeAtMost(it2, 5)
	assert.Nil(err)
	// make sure we're in the right place
	assert.Equal(expected, tokens)

	// it1 state should be different from it2 state
	assert.False(it1.State().Equal(it2.State()))

	// Make a new iterator starting from it1 state, move it to it2 position. The states
	// should then be equal
	itTmp := lex.TokeniseAt(input, it1.State())
	tokens, err = tokenizeAtMost(itTmp, 5)
	assert.Nil(err)
	assert.Equal(expected, tokens)
	assert.True(it2.State().Equal(itTmp.State()))

	// Change the text in a compatible manner
	prog2 := `int var1;
char* var2;
int var3;
	`
	input = []rune(prog2)

	// adjust the state for iterator 1 and 2 since the text changed
	st2 := it2.State()
	st2.AddToIndex(2)

	//fmt.Printf("it1 state: %s\n", it1.State())

	// make a new iterator starting at end end of line 1 (it1) and when it gets to the same point as it2 they should be equal
	itTmp = lex.TokeniseAt(input, it1.State())
	tokens, err = tokenizeAtMost(itTmp, 6)
	assert.Nil(err)
	expected = []Token{
		{Type: KeywordType, Value: []rune("char"), Start: 10, End: 14},
		{Type: Operator, Value: []rune("*"), Start: 14, End: 15},
		{Type: Text, Value: []rune(" "), Start: 15, End: 16},
		{Type: Name, Value: []rune("var2"), Start: 16, End: 20},
		{Type: Punctuation, Value: []rune(";"), Start: 20, End: 21},
		{Type: Text, Value: []rune("\n"), Start: 21, End: 22},
	}
	dumpTokens(t, tokens)
	assert.Equal(expected, tokens)
	assert.True(st2.Equal(itTmp.State()))
	//fmt.Printf("it2 state: %s\n", st2)
	//fmt.Printf("itTmp state: %s\n", itTmp.State())

}
