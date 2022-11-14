package syn

type Token struct {
	Typ   TokenType
	Value []byte
}

type TokenType int

const (
	Error TokenType = iota
	EOF
)

const (
	Keyword TokenType = 1000 + iota
	KeywordConstant
	KeywordDeclaration
	KeywordNamespace
	KeywordPseudo
	KeywordReserved
	KeywordType
)