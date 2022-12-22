package lexers

import (
	"embed"
	"github.com/jeffwilliams/syn"
	"io/fs"
)

//go:embed embedded
var embedded embed.FS

// TODO: Need to do the new Design idea from README before implementing this.

// GlobalLexerRegistry is the global LexerRegistry of Lexers.
var GlobalLexerRegistry = func() *syn.LexerRegistry {
	reg := syn.NewLexerRegistry()
	// index(reg)
	paths, err := fs.Glob(embedded, "embedded/*.xml")
	if err != nil {
		panic(err)
	}
	for _, path := range paths {
		lex, err := syn.NewLexerFromXMLFS(embedded, path)
		// TODO: save the errors here and allow retrieving them
		if err == nil {
			reg.Register(lex)
		}
	}
	return reg
}()

// Names of all lexers, optionally including aliases.
func Names(withAliases bool) []string {
	return GlobalLexerRegistry.Names(withAliases)
}

// Get a Lexer by name, alias or file extension.
func Get(name string) *syn.Lexer {
	return GlobalLexerRegistry.Get(name)
}

// MatchMimeType attempts to find a lexer for the given MIME type.
func MatchMimeType(mimeType string) *syn.Lexer {
	return GlobalLexerRegistry.MatchMimeType(mimeType)
}

// Match returns the first lexer matching filename.
func Match(filename string) *syn.Lexer {
	return GlobalLexerRegistry.Match(filename)
}

// Register a Lexer with the global registry.
/*
func Register(lexer syn.Lexer) *syn.Lexer {
	return GlobalLexerRegistry.Register(lexer)
}
*/

/*
// Analyse text content and return the "best" lexer..
func Analyse(text string) syn.Lexer {
	return GlobalLexerRegistry.Analyse(text)
}
*/

// PlaintextRules is used for the fallback lexer as well as the explicit
// plaintext lexer.
/*
func PlaintextRules() syn.Rules {
	rules := syn.NewRules()
	rules.AddState()

	return syn.Rules{rules: map[string]syn.State{
		"root": syn.State{
			name: "root",
			rules: []Rule{
				{`.+`, chroma.Text, nil},
				{`\n`, chroma.Text, nil},
			},
		},
	},
	}
}

// Fallback lexer if no other is found.
var Fallback syn.Lexer = chroma.MustNewLexer(&chroma.Config{
	Name:      "fallback",
	Filenames: []string{"*"},
	Priority:  -1,
}, PlaintextRules)
*/
