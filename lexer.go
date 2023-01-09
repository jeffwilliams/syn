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
	rules  Rules
}

func NewLexer(rules Rules) *Lexer {
	return &Lexer{
		rules: rules,
	}
}

func NewLexerFromXMLFile(xmlLexerConfigFile string) (*Lexer, error) {
	f, err := os.Open(xmlLexerConfigFile)
	if err != nil {
		return nil, err
	}

	return NewLexerFromXML(f)
}

func NewLexerFromXMLFS(fsys fs.FS, xmlLexerConfigFile string) (*Lexer, error) {
	f, err := fsys.Open(xmlLexerConfigFile)
	if err != nil {
		return nil, err
	}

	return NewLexerFromXML(f)
}

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
			rules:  NewRules(),
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
	for _, s := range lb.cfg.Rules.States {
		if s.Name == "root" {
			return nil
		}
	}
	return fmt.Errorf("No 'root' state is defined")
}

func (lb *lexerBuilder) build() error {
	for _, xmlState := range lb.cfg.Rules.States {

		seq, err := lb.ruleSequence(xmlState.Rules)
		if err != nil {
			return fmt.Errorf("For state %s: %w", xmlState.Name, err)
		}

		s := State{xmlState.Name, seq}
		lb.lexer.rules.AddState(s)
	}

	return nil
}

func (lb *lexerBuilder) ruleSequence(crs []config.Rule) ([]Rule, error) {
	rules := make([]Rule, len(crs))
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

		r := Rule{
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

	for name, state := range lb.lexer.rules.rules {

		// TODO: to reduce garbage, we could just build this when the first include is reached
		// If there are none there is no need to make a new slice.
		newSeq := make([]Rule, 0, len(state.rules))

		for _, rule := range state.rules {
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

		lb.lexer.rules.rules[name] = State{name, newSeq}
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
