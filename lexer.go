package syn

import (
	"fmt"
	"github.com/jeffwilliams/syn/internal/config"
	"os"
	"regexp"
)

type Lexer struct {
	state State
	text  []byte
	rules Rules
}

func NewLexer(text []byte, rules Rules) *Lexer {
	return &Lexer{
		state: State{stack: NewStack()},
		text:  text,
		rules: rules,
	}
}

func NewLexerFromXML(textToHighlight []byte, xmlLexerConfigFile string) (*Lexer, error) {
	// TODO: refactor this
	f, err := os.Open(xmlLexerConfigFile)
	if err != nil {
		return nil, err
	}

	lexModel, err := config.DecodeLexer(f)
	if err != nil {
		return nil, err
	}

	bld := newLexerBuilder(lexModel)
	return bld.Build()
}

func (l *Lexer) PushState(state string) error {
	s := l.rules.RuleSequenceForState(state)
	if s == nil {
		return fmt.Errorf("No state %s", state)
	}
	l.state.stack.Push(s)
	return nil
}

func (l *Lexer) Next() ([]Token, error) {
	l.pushRootStateIfNeeded()

	if l.state.index >= len(l.text) {
		return []Token{{Typ: EOFType, Value: nil}}, nil
	}

	rules := l.state.stack.Top()
	match, rule := rules.match(l.text[l.state.index:])
	if match == nil {
		return []Token{{Error, nil}}, nil
	}

	// TODO: carriage returns?

	toks, err := l.tokensOfMatch(match, rule)
	if err != nil {
		return nil, err
	}

	l.state.index = match[1]

	l.handleRuleState(rule)

	return toks, nil
}

func (l *Lexer) pushRootStateIfNeeded() {
	fmt.Printf("pushRootStateIfNeeded called\n")
	if l == nil {
		fmt.Printf("lexer is nil\n")
	}
	if l.state.stack == nil {
		fmt.Printf("stack is nil\n")
	}

	if l.state.stack.Len() == 0 {
		s := l.rules.RuleSequenceForState("root")
		if s != nil {
			l.state.stack.Push(s)
		}
	}
}

func (l *Lexer) tokensOfMatch(match []int, rule *Rule) ([]Token, error) {
	if rule.byGroups != nil {
		if len(match)/2 < len(rule.byGroups) {
			return nil, fmt.Errorf("Rule has more actions in ByGroups than there are groups in the regular expression")
		}

		toks := []Token{}

		for i, g := range rule.byGroups {
			groupIndex := (i + 1) * 2

			if g.IsUseSelf() {
				lex := NewLexer(l.matchText(match[groupIndex:]), l.rules)
				lex.PushState(g.useSelfState)
				lex.LexInto(&toks)
			} else {
				// TODO make a token here
				t := Token{Typ: g.tok, Value: l.matchText(match[groupIndex:])}
				toks = append(toks, t)
			}
		}

		return toks, nil
	}

	return []Token{{Typ: rule.tok, Value: l.matchText(match)}}, nil
}

func (l *Lexer) Lex() []Token {
	toks := []Token{}
	l.LexInto(&toks)
	return toks
}

func (l *Lexer) LexInto(toks *[]Token) error {
	for {
		t, err := l.Next()
		if err != nil {
			return err
		}
		*toks = append(*toks, t...)
		if l.containsErrorOrEof(*toks) {
			return nil
		}
	}
}

func (l *Lexer) containsErrorOrEof(toks []Token) bool {
	for _, t := range toks {
		if t.Typ == Error || t.Typ == EOFType {
			return true
		}
	}
	return false
}

func (l *Lexer) handleRuleState(rule *Rule) {
	if rule.popDepth == 0 && rule.pushState == "" {
		return
	}

	if rule.popDepth > 0 {
		l.state.stack.Pop(rule.popDepth)
		return
	}

	s, ok := l.rules.rules[rule.pushState]
	if !ok {
		msg := fmt.Sprintf("syn.Lexer: a rule refers to a state %s that doesn't exist", rule.pushState)
		panic(msg)
	}
	l.state.stack.Push(s)
}

func (l *Lexer) matchText(match []int) []byte {
	return l.text[match[0]:match[1]]
}

func (l *Lexer) State() State {
	return l.state
}

func (l *Lexer) SetState(s State) {
	l.state = s
}

// State represents the state of the Lexer at some intermediate position in the lexing.
// It determines what token should be matched next based on what has aleady been processed up to
// a certain byte-position in the input text. It can be used to restart lexing from that same point
// in the text.
type State struct {
	stack *Stack
	index int
}

type Action struct {
	typ       ActionType
	tokenType TokenType
}

type ActionType int

const (
	Pop ActionType = iota
	EmitToken
)

type lexerBuilder struct {
	cfg   *config.Lexer
	lexer *Lexer
}

func newLexerBuilder(cfg *config.Lexer) lexerBuilder {
	return lexerBuilder{
		cfg: cfg,
		lexer: &Lexer{
			rules: NewRules(),
			state: State{stack: NewStack()},
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
			return err
		}

		lb.lexer.rules.AddRuleSequence(xmlState.Name, seq)
	}

	return nil
}

func (lb *lexerBuilder) ruleSequence(crs []config.Rule) (RuleSequence, error) {
	rules := make([]Rule, len(crs))
	for i, cr := range crs {
		err := lb.checkRule(&cr)
		if err != nil {
			return nil, err
		}

		re, err := regexp.Compile(cr.Pattern)
		if err != nil {
			return nil, err
		}

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
	return RuleSequence(rules), nil
}

func (lb *lexerBuilder) checkRule(r *config.Rule) error {
	// A rule may have only of the following sets:
	// 1. A token and _either_ a push or pop
	// 2. An Include
	// 3. A ByGroups

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

	for i, seq := range lb.lexer.rules.rules {

		// TODO: to reduce garbage, we could just build this when the first include is reached
		// If there are none there is no need to make a new slice.
		newSeq := RuleSequence(make([]Rule, 0, len(seq)))

		for _, rule := range seq {
			if rule.include != "" {
				includeSeq := lb.lexer.rules.RuleSequenceForState(rule.include)
				if includeSeq == nil {
					return fmt.Errorf("A rule includes the state named '%s' but there is no such state in the lexer", rule.include)
				}

				for _, e := range includeSeq {
					newSeq = append(newSeq, e)
				}
				continue
			}

			newSeq = append(newSeq, rule)
		}

		lb.lexer.rules.rules[i] = newSeq
	}

	return nil
}
