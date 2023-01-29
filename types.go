package syn

//go:generate go run github.com/dmarkham/enumer -text -type TokenType

// TokenType is the type of token to highlight.
type TokenType int

// Set of TokenTypes.
//
// Categories of types are grouped in ranges of 1000, while sub-categories are in ranges of 100. For
// example, the literal category is in the range 3000-3999. The sub-category for literal strings is
// in the range 3100-3199.

// Meta token types.
const (
	// Default background style.
	Background TokenType = -1 - iota
	// PreWrapper style.
	PreWrapper
	// Line style.
	Line
	// Line numbers in output.
	LineNumbers
	// Line numbers in output when in table.
	LineNumbersTable
	// Line higlight style.
	LineHighlight
	// Line numbers table wrapper style.
	LineTable
	// Line numbers table TD wrapper style.
	LineTableTD
	// Line number links.
	LineLink
	// Code line wrapper style.
	CodeLine
	// Input that could not be tokenised.
	Error
	// Other is used by the Delegate lexer to indicate which tokens should be handled by the delegate.
	Other
	// No highlighting.
	None
	// Used as an EOF marker / nil token
	EOFType TokenType = 0
)

// Keywords.
const (
	Keyword TokenType = 1000 + iota
	KeywordConstant
	KeywordDeclaration
	KeywordNamespace
	KeywordPseudo
	KeywordReserved
	KeywordType
)

// Names.
const (
	Name TokenType = 2000 + iota
	NameAttribute
	NameBuiltin
	NameBuiltinPseudo
	NameClass
	NameConstant
	NameDecorator
	NameEntity
	NameException
	NameFunction
	NameFunctionMagic
	NameKeyword
	NameLabel
	NameNamespace
	NameOperator
	NameOther
	NamePseudo
	NameProperty
	NameTag
	NameVariable
	NameVariableAnonymous
	NameVariableClass
	NameVariableGlobal
	NameVariableInstance
	NameVariableMagic
)

// Literals.
const (
	Literal TokenType = 3000 + iota
	LiteralDate
	LiteralOther
)

// Strings.
const (
	LiteralString TokenType = 3100 + iota
	LiteralStringAffix
	LiteralStringAtom
	LiteralStringBacktick
	LiteralStringBoolean
	LiteralStringChar
	LiteralStringDelimiter
	LiteralStringDoc
	LiteralStringDouble
	LiteralStringEscape
	LiteralStringHeredoc
	LiteralStringInterpol
	LiteralStringName
	LiteralStringOther
	LiteralStringRegex
	LiteralStringSingle
	LiteralStringSymbol
)

// Literals.
const (
	LiteralNumber TokenType = 3200 + iota
	LiteralNumberBin
	LiteralNumberFloat
	LiteralNumberHex
	LiteralNumberInteger
	LiteralNumberIntegerLong
	LiteralNumberOct
)

// Operators.
const (
	Operator TokenType = 4000 + iota
	OperatorWord
)

// Punctuation.
const (
	Punctuation TokenType = 5000 + iota
)

// Comments.
const (
	Comment TokenType = 6000 + iota
	CommentHashbang
	CommentMultiline
	CommentSingle
	CommentSpecial
)

// Preprocessor "comments".
const (
	CommentPreproc TokenType = 6100 + iota
	CommentPreprocFile
)

// Generic tokens.
const (
	Generic TokenType = 7000 + iota
	GenericDeleted
	GenericEmph
	GenericError
	GenericHeading
	GenericInserted
	GenericOutput
	GenericPrompt
	GenericStrong
	GenericSubheading
	GenericTraceback
	GenericUnderline
)

// Text.
const (
	Text TokenType = 8000 + iota
	TextWhitespace
	TextSymbol
	TextPunctuation
)

// Aliases.
const (
	Whitespace = TextWhitespace

	Date = LiteralDate

	String          = LiteralString
	StringAffix     = LiteralStringAffix
	StringBacktick  = LiteralStringBacktick
	StringChar      = LiteralStringChar
	StringDelimiter = LiteralStringDelimiter
	StringDoc       = LiteralStringDoc
	StringDouble    = LiteralStringDouble
	StringEscape    = LiteralStringEscape
	StringHeredoc   = LiteralStringHeredoc
	StringInterpol  = LiteralStringInterpol
	StringOther     = LiteralStringOther
	StringRegex     = LiteralStringRegex
	StringSingle    = LiteralStringSingle
	StringSymbol    = LiteralStringSymbol

	Number            = LiteralNumber
	NumberBin         = LiteralNumberBin
	NumberFloat       = LiteralNumberFloat
	NumberHex         = LiteralNumberHex
	NumberInteger     = LiteralNumberInteger
	NumberIntegerLong = LiteralNumberIntegerLong
	NumberOct         = LiteralNumberOct
)

func (t TokenType) Parent() TokenType {
	if t%100 != 0 {
		return t / 100 * 100
	}
	if t%1000 != 0 {
		return t / 1000 * 1000
	}
	return 0
}

func (t TokenType) Category() TokenType {
	return t / 1000 * 1000
}

func (t TokenType) SubCategory() TokenType {
	return t / 100 * 100
}

func (t TokenType) InCategory(other TokenType) bool {
	return t/1000 == other/1000
}

func (t TokenType) InSubCategory(other TokenType) bool {
	return t/100 == other/100
}
