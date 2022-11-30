package syn

import (
	"fmt"
	"os"
	"regexp"

	"github.com/jeffwilliams/syn/internal/config"
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

func (l *Lexer) Next() ([]Token, error) {
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

func (l *Lexer) tokensOfMatch(match []int, rule *Rule) ([]Token, error) {
	if rule.byGroups != nil {
		if len(match)/2 < len(rule.byGroups) {
			return nil, fmt.Errorf("Rule has more actions in ByGroups than there are groups in the regular expression")
		}

		toks := []Token{}

		for i, g := range rule.byGroups {
			groupIndex := (i + 1) * 2

			if g.useSelf {
				lex := NewLexer(l.matchText(match[groupIndex:]), l.rules)
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

func (l *Lexer) LexInto(toks *[]Token) {
	for {
		t := l.Next()
		*toks = append(*toks, t)
		if t.Typ == Error || t.Typ == EOFType {
			return
		}
	}
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

type Stack struct {
	data []RuleSequence
}

func NewStack() *Stack {
	return &Stack{
		data: make([]RuleSequence, 0),
	}
}

func (s Stack) Push(list RuleSequence) {
}

func (s Stack) Pop(count int) (list RuleSequence) {
	return nil
}

func (s Stack) Top() (list RuleSequence) {
	return nil
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
		},
	}
}

func (lb *lexerBuilder) Build() (*Lexer, error) {

	err := lb.build()
	if err != nil {
		return nil, err
	}

	lb.resolveIncludes()

	return lb.lexer, nil
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

		typ, err := TokenTypeString(cr.Token.Type)
		if err != nil {
			return nil, err
		}

		r := Rule{
			pattern: re,
			tok:     typ,
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
