package syn

import "fmt"

// Token is a single token returned during iteration when lexing.
type Token struct {
	Type       TokenType
	Value      []rune
	Start, End int
}

// String returns a textual description of the fields of the token. To get the text of the token use Value instead.
func (t Token) String() string {
	return fmt.Sprintf("token: Type: %s Value: '%s' Start: %d End: %d",
		t.Type, string(t.Value), t.Start, t.End)
}

// Length returns the length of the token.
func (t Token) Length() int {
	return t.End - t.Start
}
