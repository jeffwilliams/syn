package syn

import "fmt"

type Token struct {
	Typ        TokenType
	Value      []rune
	Start, End int
}

func (t Token) String() string {
	return fmt.Sprintf("token: Type: %s Value: '%s' len: %d", t.Typ, string(t.Value), len(t.Value))
}
