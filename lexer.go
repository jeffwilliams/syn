// Package syn implements a syntax highlighter meant to be used in text editors.
//
// The syn package exports a Lexer type which can be used to lex text for a specific language
// and return an Iterator that can iterate over the Tokens of the text. The state of the iteration
// can be saved and later used to restart iteration from the saved point. This is useful in text editors
// to re-highlight a subset of the text that has been modified rather than the entire text by iterating
// starting with the saved state rather than from the beginning of the text.
//
// Lexers are normally created using the lexers subpackage. For example:
//
//   import "github.com/jeffwilliams/syn/lexers"
//
//   lexer = lexers.Get("Go")
package syn

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/jeffwilliams/syn/internal/config"
)

type Lexer struct {
	config *config.Lexer
	rules  rules
}

func newLexer(r rules) *Lexer {
	return &Lexer{
		rules: r,
	}
}

// NewLexerFromXML creates a new lexer given an XML file containing a definition of a lexer.
func NewLexerFromXMLFile(xmlLexerConfigFile string) (*Lexer, error) {
	f, err := os.Open(xmlLexerConfigFile)
	if err != nil {
		return nil, err
	}

	return NewLexerFromXML(f)
}

// NewLexerFromXML creates a new lexer given an XML file containing a definition of a lexer. The file is opened
// using the specified FS.
func NewLexerFromXMLFS(fsys fs.FS, xmlLexerConfigFile string) (*Lexer, error) {
	f, err := fsys.Open(xmlLexerConfigFile)
	if err != nil {
		return nil, err
	}

	return NewLexerFromXML(f)
}

// NewLexerFromXML creates a new lexer given an XML definition of a lexer.
func NewLexerFromXML(rdr io.Reader) (*Lexer, error) {
	lexModel, err := config.DecodeLexer(rdr)
	if err != nil {
		return nil, err
	}

	bld := newLexerBuilder(lexModel)
	lex, err := bld.Build()
	if err != nil {
		return nil, err
	}

	return lex, nil

}

func (l *Lexer) Tokenise(text []rune) Iterator {
	stripped, offsetMap := ensureLF(text)
	innerIter := newIterator(stripped, l.rules)
	outerIter := adjustForLF(text, innerIter, offsetMap.iterator())
	return outerIter
}

func (l *Lexer) cfg() *config.Lexer {
	return l.config
}

type lexerBuilder struct {
	cfg   *config.Lexer
	lexer *Lexer
}

func newLexerBuilder(cfg *config.Lexer) lexerBuilder {
	return lexerBuilder{
		cfg: cfg,
		lexer: &Lexer{
			rules:  newRules(),
			config: cfg,
		},
	}
}

func (lb *lexerBuilder) Build() (*Lexer, error) {

	err := lb.validate()
	if err != nil {
		return nil, err
	}

	err = lb.build()
	if err != nil {
		return nil, err
	}

	lb.resolveIncludes()

	return lb.lexer, nil
}

func (lb *lexerBuilder) validate() error {
	foundRoot := false
	for _, s := range lb.cfg.Rules.States {
		if s.Name == "root" {
			foundRoot = true
		}
	}

	if !foundRoot {
		return fmt.Errorf("No 'root' state is defined")
	}

	var missing []string

	stateNames := map[string]struct{}{}
	for _, state := range lb.cfg.Rules.States {
		stateNames[state.Name] = struct{}{}
	}

	for _, state := range lb.cfg.Rules.States {
		for _, rule := range state.Rules {
			if rule.Push == nil || rule.Push.State == "" {
				continue
			}

			if _, ok := stateNames[rule.Push.State]; !ok {
				missing = append(missing, rule.Push.State)
			}
		}
	}

	return lb.makeMissingError(missing)
}

func (r lexerBuilder) makeMissingError(missing []string) error {
	if missing == nil || len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("The following states are referred to from rules, but aren't defined: %v\n",
		missing)
}

func (lb *lexerBuilder) build() error {
	for _, xmlState := range lb.cfg.Rules.States {

		seq, err := lb.ruleSequence(xmlState.Rules)
		if err != nil {
			return fmt.Errorf("For state %s: %w", xmlState.Name, err)
		}

		s := state{xmlState.Name, seq}
		lb.lexer.rules.AddState(s)
	}

	return nil
}

func (lb *lexerBuilder) ruleSequence(crs []config.Rule) ([]rule, error) {
	rules := make([]rule, len(crs))
	for i, cr := range crs {
		err := lb.checkRule(&cr)
		if err != nil {
			return nil, fmt.Errorf("rule index %d: %w", i, err)
		}

		pat := `\A` + cr.Pattern
		re, err := regexp2.Compile(pat, regexp2.None)
		if err != nil {
			return nil, err
		}
		re.MatchTimeout = time.Millisecond * 250

		r := rule{
			pattern: re,
		}

		if cr.Token != nil {
			typ, err := TokenTypeString(cr.Token.Type)
			if err != nil {
				return nil, err
			}
			r.tok = typ
		}

		if cr.Pop != nil {
			r.popDepth = cr.Pop.Depth
		}

		if cr.Push != nil {
			r.pushState = cr.Push.State
		}

		if cr.Include != nil {
			r.include = cr.Include.State
		}

		if cr.ByGroups != nil {
			for _, e := range cr.ByGroups.ByGroupsElements {
				ge := byGroupElement{}
				switch v := e.V.(type) {
				case *config.Token:
					typ, err := TokenTypeString(v.Type)
					if err != nil {
						return nil, err
					}
					ge.tok = typ
				case *config.UsingSelf:
					ge.useSelfState = v.State
				}
				r.byGroups = append(r.byGroups, ge)
			}
		}

		rules[i] = r
	}
	return rules, nil
}

func (lb *lexerBuilder) checkRule(r *config.Rule) error {
	// A rule may have only of the following sets:
	// 1. A token and _either_ a push or pop
	// 2. An Include
	// 3. A ByGroups

	if r.Pattern == "" && r.Push == nil && r.Include == nil {
		return fmt.Errorf("Rule has no pattern and no push statement. This is not supported.")
	}

	if r.Pop != nil && r.Push != nil {
		return fmt.Errorf("Rule contains both a push and a pop")
	}

	if r.Token != nil {
		if r.Include != nil {
			return fmt.Errorf("a rule has both a Token and an Include")
		}
		if r.ByGroups != nil {
			return fmt.Errorf("a rule has both a Token and a ByGroups")
		}
	}

	if r.Include != nil {
		if r.ByGroups != nil {
			return fmt.Errorf("a rule has both an Include and a ByGroups")
		}
	}

	return nil
}

func (lb *lexerBuilder) resolveIncludes() error {

	for name, st := range lb.lexer.rules.rules {

		// TODO: to reduce garbage, we could just build this when the first include is reached
		// If there are none there is no need to make a new slice.
		newSeq := make([]rule, 0, len(st.rules))

		for _, rule := range st.rules {
			if rule.include != "" {
				includeState, ok := lb.lexer.rules.Get(rule.include)
				if !ok {
					return fmt.Errorf("A rule includes the state named '%s' but there is no such state in the lexer", rule.include)
				}

				for _, e := range includeState.rules {
					newSeq = append(newSeq, e)
				}
				continue
			}

			newSeq = append(newSeq, rule)
		}

		lb.lexer.rules.rules[name] = state{name, newSeq}
	}

	return nil
}

type prioritisedLexers []*Lexer

func (l prioritisedLexers) Len() int      { return len(l) }
func (l prioritisedLexers) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l prioritisedLexers) Less(i, j int) bool {
	ip := l[i].cfg().Config.Priority
	if ip == 0 {
		ip = 1
	}
	jp := l[j].cfg().Config.Priority
	if jp == 0 {
		jp = 1
	}
	return ip > jp
}
