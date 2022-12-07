package syn

import (
	"fmt"
	"os"

	"github.com/dlclark/regexp2"
	"github.com/jeffwilliams/syn/internal/config"
)

type Lexer struct {
	state LexerState
	text  []rune
	rules Rules
	depth int
}

func NewLexer(text []rune, rules Rules) *Lexer {
	return &Lexer{
		state: LexerState{stack: NewStack()},
		text:  text,
		rules: rules,
	}
}

func NewLexerFromXML(textToHighlight []rune, xmlLexerConfigFile string) (*Lexer, error) {
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
	lex, err := bld.Build()
	if err != nil {
		return nil, err
	}

	lex.text = textToHighlight
	return lex, nil
}

func (l *Lexer) PushState(state string) error {
	s, ok := l.rules.Get(state)
	if !ok {
		return fmt.Errorf("No state %s", state)
	}
	l.state.stack.Push(s)
	return nil
}

func (l *Lexer) Next() ([]Token, error) {
	l.pushRootStateIfNeeded()

	if l.state.index >= len(l.text) {
		debugf("Lexer.Next(%d): Current index %d is past the end of the text. Text has length %d",
			l.depth, l.state.index, len(l.text))
		return []Token{{Typ: EOFType, Value: nil}}, nil
	}

	rules := l.state.stack.Top()
	debugf("Lexer.Next(%d): Matching top state %s", l.depth, rules.name)
	match, rule := rules.match(l.text[l.state.index:])
	if match == nil {
		debugf("Lexer.Next(%d): No rule in the rule sequence matched", l.depth)
		return []Token{{Error, nil}}, nil
	}

	// TODO: carriage returns?

	toks, err := l.tokensOfMatch(match, rule)
	if err != nil {
		debugf("Lexer.Next(%d): got an error creating tokens from the match: %v", l.depth, err)
		return nil, err
	}

	g := match.GroupByNumber(0)
	debugf("Lexer.Next(%d): Moving index from %d to %d (some text there is: '%s')", l.depth, l.state.index, l.state.index+g.Length,
		aLittleText(l.text, l.state.index+g.Length))
	l.state.index += g.Length

	l.handleRuleState(rule)

	debugf("Lexer.Next(%d): returning %d tokens", l.depth, len(toks))
	return toks, nil
}

func aLittleText(r []rune, index int) string {
	if index >= len(r) {
		return ""
	}

	end := index + 5
	if end >= len(r) {
		end = len(r)
	}

	return string(r[index:end])

}

func (l *Lexer) pushRootStateIfNeeded() {
	if l.state.stack.Len() == 0 {
		debugf("Lexer.pushRootStateIfNeeded(%d): Pushing root state", l.depth)

		s, ok := l.rules.Get("root")
		if ok {
			l.state.stack.Push(s)
		} else {
			debugf("No root state found in lexer")
		}
	}
}

func (l *Lexer) tokensOfMatch(match *regexp2.Match, rule *Rule) ([]Token, error) {
	if rule.byGroups == nil {
		if rule.tok == 0 {
			debugf("Lexer.tokensOfMatch(%d): rule provides no token\n", l.depth)
			return []Token{}, nil
		}

		debugf("Lexer.tokensOfMatch(%d): returning token for entire match\n", l.depth)
		// Use entire match
		return []Token{{Typ: rule.tok, Value: l.groupText(match.GroupByNumber(0))}}, nil
	}

	if match.GroupCount() < len(rule.byGroups) {
		return nil, fmt.Errorf("Rule has more actions in ByGroups than there are groups in the regular expression")
	}

	toks := []Token{}

	debugf("Lexer.tokensOfMatch(%d): rule specified to classify by groups\n", l.depth)
	for i, g := range rule.byGroups {
		//groupIndex := (i + 1) * 2

		groupText := l.groupText(match.GroupByNumber(i + 1))
		if g.IsUseSelf() {
			debugf("Lexer.tokensOfMatch(%d): bygroups %d is a use-self. Creating sub lexer\n", l.depth, i)
			lex := NewLexer(groupText, l.rules)
			lex.depth = l.depth + 1
			lex.PushState(g.useSelfState)
			lex.LexInto(&toks)
			// Remove the final EOF token
			toks = toks[:len(toks)-1]
		} else {
			debugf("Lexer.tokensOfMatch(%d): bygroups %d: appending token\n", l.depth, i)
			t := Token{Typ: g.tok, Value: groupText}
			toks = append(toks, t)
		}
	}

	return toks, nil

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
			debugf("Lexer.LexInto: tokens returned by Next contain EOF or Error token so returning")
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
		debugf("Lexer.handleRuleState(%d): Popping %d states", l.depth, rule.popDepth)
		l.state.stack.Pop(rule.popDepth)
		return
	}

	s, ok := l.rules.rules[rule.pushState]
	if !ok {
		msg := fmt.Sprintf("syn.Lexer: a rule refers to a state %s that doesn't exist", rule.pushState)
		panic(msg)
	}
	debugf("Lexer.handleRuleState(%d): pushing state %s", l.depth, rule.pushState)
	l.state.stack.Push(s)
}

func (l *Lexer) groupText(g *regexp2.Group) []rune {
	text := l.text[l.state.index:]
	return text[g.Index : g.Index+g.Length]
}

func (l *Lexer) LexerState() LexerState {
	return l.state
}

func (l *Lexer) SetLexerState(s LexerState) {
	l.state = s
}

// LexerState represents the state of the Lexer at some intermediate position in the lexing.
// It determines what token should be matched next based on what has aleady been processed up to
// a certain byte-position in the input text. It can be used to restart lexing from that same point
// in the text.
type LexerState struct {
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
			state: LexerState{stack: NewStack()},
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

		re, err := regexp2.Compile(cr.Pattern, regexp2.None)
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
