package syn

import "fmt"

type Token struct {
	Typ        TokenType
	Value      []rune
	Start, End int
}

func (t Token) String() string {
	return fmt.Sprintf("token: Type: %s Value: '%s' Start: %d End: %d",
		t.Typ, string(t.Value), t.Start, t.End)
}

func (t Token) Length() int {
	return t.End - t.Start
}
