// Lexers contains lexers for the syn package and methods for creating syn Lexers
package lexers

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/jeffwilliams/syn"
)

//go:embed embedded
var embedded embed.FS

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
		if err != nil {
			GlobalLexerLoadErrors = append(GlobalLexerLoadErrors, fmt.Errorf("Error loading lexer %s: %s", path, err))
			continue
		}
		reg.Register(lex)

	}
	return reg
}()

var GlobalLexerLoadErrors []error

// Names of all lexers, optionally including aliases.
func Names(withAliases bool) []string {
	return GlobalLexerRegistry.Names(withAliases)
}

// Get a Lexer by name, alias or file extension. Returns nil when no matching lexer is found.
func Get(name string) *syn.Lexer {
	return GlobalLexerRegistry.Get(name)
}

// MatchMimeType attempts to find a lexer for the given MIME type. Returns nil when no matching lexer is found.
func MatchMimeType(mimeType string) *syn.Lexer {
	return GlobalLexerRegistry.MatchMimeType(mimeType)
}

// Match returns the first lexer matching filename. Returns nil when no matching lexer is found.
func Match(filename string) *syn.Lexer {
	return GlobalLexerRegistry.Match(filename)
}
