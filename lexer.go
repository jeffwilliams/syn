package syn

import (
	"fmt"
	"os"

	"github.com/dlclark/regexp2"
	"github.com/jeffwilliams/syn/internal/config"
)

type Lexer struct {
	// state stores the state of this Lexer at the current point in the lexing.
	// Element 0 is the state of this lexer, and 1 and above are the state of
	// sublexers, if any, that are processing a subset of the text. LexerState[1]
	// is the state for the lexer at depth 1, [2] for depth 2, and so on.
	state     LexerState
	sublexers []*Lexer
	rules     Rules
	depth     int
}

func NewLexer(text []rune, rules Rules) *Lexer {
	return &Lexer{
		state: LexerState{text: text, stack: NewStack()},
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

	lex.state.text = textToHighlight
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

func (l *Lexer) Next() (Token, error) {
	l.pushRootStateIfNeeded()

	switch l.state.stage {
	case stageReadyToMatch:
		return l.nextInReadyToMatchStage()
	case stageWithinGroups:
		return l.nextInWithinGroupsStage()
	case stageRunningSublexer:
		return l.nextInSublexer()
	default:
		return Token{}, fmt.Errorf("Unsupported lexer stage %d", l.state.stage)
	}
}

func (l *Lexer) nextInReadyToMatchStage() (tok Token, err error) {
	if l.state.index >= len(l.state.text) {
		debugf("Lexer.Next(%d): Current index %d is past the end of the text. Text has length %d. Returning EOFType",
			l.depth, l.state.index, len(l.state.text))
		return Token{Typ: EOFType, Value: nil}, nil
	}

	state := l.state.stack.Top()
	debugf("Lexer.Next(%d): Matching a full rule in top state %s", l.depth, state.name)
	match, rule := state.match(l.state.text[l.state.index:])
	if match == nil {
		debugf("Lexer.Next(%d): No rule in the rule sequence matched", l.depth)
		return Token{Typ: Error, Value: nil}, nil
	}

	// TODO: carriage returns?

	if rule.byGroups != nil {
		l.prepareToIterateGroups(rule, match)
		return l.Next()
	}

	if rule.tok == 0 {
		debugf("Lexer.Next(%d): rule provides no token\n", l.depth)
		tok = Token{}
	} else {
		debugf("Lexer.Next(%d): will return token for entire match\n", l.depth)
		// Use entire match
		tok = l.tokenOfEntireMatch(rule.tok, match)
		g := match.GroupByNumber(0)
		debugf("Lexer.Next(%d): Moving index from %d to %d (some text there is: '%s')", l.depth, l.state.index, l.state.index+g.Length,
			aLittleText(l.state.text, l.state.index+g.Length))
		l.state.index += g.Length
	}
	l.handleRuleState(rule)

	if rule.tok == 0 {
		debugf("Lexer.Next(%d): recursing to generate token\n", l.depth)
		return l.Next()
	}

	debugf("Lexer.Next(%d): returning token %s", l.depth, tok)
	return
}

func (l *Lexer) prepareToIterateGroups(matchingRule *Rule, match *regexp2.Match) {
	l.state.rule = matchingRule
	l.setCapturesFromMatch(match)
	l.state.groupIndex = 0
	l.state.stage = stageWithinGroups
	l.state.byGroups = matchingRule.byGroups
}

func (l *Lexer) setCapturesFromMatch(match *regexp2.Match) {
	l.state.groups = make([]capture, match.GroupCount())
	for i, g := range match.Groups() {
		debugf("Lexer.Next(%d): group %d in match is at %d of length %d", l.depth, i, g.Index, g.Length)
		l.state.groups[i].start = g.Index
		l.state.groups[i].length = g.Length
	}
}

func (l *Lexer) nextInWithinGroupsStage() (tok Token, err error) {
	debugf("Lexer.nextInWithinGroupsStage(%d): Will return the next group with index %d (%d/%d)", l.depth, l.state.groupIndex, l.state.groupIndex+1, len(l.state.byGroups))

	byGroup := l.state.byGroups[l.state.groupIndex]
	capture := l.state.groups[l.state.groupIndex+1]

	text := l.state.text[l.state.index:]
	groupText := text[capture.start:capture.end()]
	if byGroup.IsUseSelf() {
		debugf("Lexer.nextInWithinGroupsStage(%d): bygroups %d is a use-self. Creating sub lexer\n", l.depth, l.state.groupIndex)
		l.prepareToUseSublexer(groupText, &capture, &byGroup)
		return l.Next()
	}

	start, end := l.boundsOfGroup(capture.start, capture.length)
	debugf("Lexer.nextInWithinGroupsStage(%d): bygroups %d: returning token\n", l.depth, l.state.groupIndex)
	tok = Token{Typ: byGroup.tok, Value: groupText, Start: start, End: end}

	l.state.groupIndex++

	if l.state.groupIndex >= len(l.state.byGroups) {
		debugf("Lexer.nextInWithinGroupsStage(%d): reached end of the groups, will switch to full match stage\n", l.depth)
		l.handleRuleState(l.state.rule)
		l.state.stage = stageReadyToMatch
		l.state.index += l.state.groups[0].length // Move past the length of the match
		l.clearGroupIterationInfo()
	}

	return tok, nil
}

func (l *Lexer) prepareToUseSublexer(groupText []rune, capture *capture, byGroup *byGroupElement) {
	lex := NewLexer(groupText, l.rules)
	lex.setOffset(l.state.index + capture.start)
	lex.depth = l.depth + 1
	lex.PushState(byGroup.useSelfState)
	l.state.stage = stageRunningSublexer
	l.sublexers = append(l.sublexers, lex)
}

func (l *Lexer) completeGroupIteration() {
	l.handleRuleState(l.state.rule)
	l.state.stage = stageReadyToMatch
	l.state.index += l.state.groups[0].length // Move past the length of the match
	l.clearGroupIterationInfo()
}

func (l *Lexer) clearGroupIterationInfo() {
	l.state.groups = l.state.groups[:0]
	l.state.groupIndex = 0
	l.state.byGroups = nil
	l.state.rule = nil
}

func (l *Lexer) nextInSublexer() (tok Token, err error) {
	tok, err = l.sublexers[len(l.sublexers)-1].Next()
	if err != nil {
		// On an error, we need to clean up all sublexers.
		// Each parent lexer will clean up it's first child lexer.
		l.sublexers = l.sublexers[:len(l.sublexers)-1]
		return
	}

	if tok.Typ == EOFType {
		debugf("Lexer.nextInSublexer(%d): sublexer completed\n", l.depth)
		// Sublexer completed. Are we still in gruops?
		l.sublexers = l.sublexers[:len(l.sublexers)-1]
		l.state.stage = stageWithinGroups
		l.state.groupIndex++
		debugf("Lexer.nextInSublexer(%d): Setting groupindex to %d (%d/%d)\n", l.depth, l.state.groupIndex, l.state.groupIndex+1, len(l.state.byGroups))
		if l.state.groupIndex > len(l.state.byGroups) {
			debugf("Lexer.nextInSublexer(%d): Reached end of groups, will switch to full match stage\n", l.depth)
			l.completeGroupIteration()
		}
		return l.Next()
	}

	return
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
	if l == nil {
		debugf("Lexer.pushRootStateIfNeeded(%d): lexer is nil", l.depth)
	}
	if l.state.stack == nil {
		debugf("Lexer.pushRootStateIfNeeded(%d): stack is nil", l.depth)
	}

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

func (l *Lexer) tokenOfEntireMatch(typ TokenType, match *regexp2.Match) Token {
	s, e := l.boundsOfCapture(&match.Group.Capture)
	return Token{Typ: typ, Value: l.groupText(match.GroupByNumber(0)), Start: s, End: e}
}

func (l *Lexer) boundsOfCapture(match *regexp2.Capture) (start, end int) {
	return l.state.index + l.state.offset + match.Index,
		l.state.index + l.state.offset + match.Index + match.Length
}

func (l *Lexer) boundsOfGroup(index, length int) (start, end int) {
	return l.state.index + l.state.offset + index,
		l.state.index + l.state.offset + index + length
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
		*toks = append(*toks, t)
		if t.Typ == Error || t.Typ == EOFType {
			debugf("Lexer.LexInto: token returned by Next is EOF or Error so returning")
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
	text := l.state.text[l.state.index:]
	return text[g.Index : g.Index+g.Length]
}

func (l *Lexer) State() []LexerState {
	states := make([]LexerState, len(l.sublexers)+1)
	states[0] = l.state
	states[0].stack = l.state.stack.Clone()
	for i, sl := range l.sublexers {
		states[i+1] = sl.state
		states[i+1].stack = sl.state.stack.Clone()
	}

	return states
}

// TODO: create the sublexers we need as well.
func (l *Lexer) SetState(s []LexerState) {
	// The lexers and sublexers all use the same rules because the only way to make
	// a sublexer is through usingself (for now).
	// So we can just clone the rules from the base lexer to all the sublexers.
	l.state = s[0]

	l.sublexers = make([]*Lexer, len(s)-1)
	for i, state := range s[1:] {
		text := state.text

		l.sublexers[i] = NewLexer(text, l.rules)
		l.depth = i + 1
		l.sublexers[i].state = state
	}
}

// setOffset sets a number that is added to the Start and End of each Token
// Next() produces before it is returned.
func (l *Lexer) setOffset(i int) {
	l.state.offset = i
}

type stage int

const (
	stageReadyToMatch = iota
	stageWithinGroups
	stageRunningSublexer
)

// LexerState represents the state of the Lexer at some intermediate position in the lexing.
// It determines what token should be matched next based on what has aleady been processed up to
// a certain byte-position in the input text. It can be used to restart lexing from that same point
// in the text.
type LexerState struct {
	stack  *Stack
	index  int
	text   []rune
	stage  stage
	offset int

	groups     []capture        // Groups from the last match. Used when we need to iterate over bygroups. Each entry is a start/end of group.
	groupIndex int              // index for current group we are iterating over
	byGroups   []byGroupElement // The "by groups" items defined in the rule that specify how to handle each group from the match
	rule       *Rule            // Rule we are matching the groups for
}

type capture struct {
	start, length int
}

func (c capture) end() int {
	return c.start + c.length
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
