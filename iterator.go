package syn

import (
	"fmt"

	"github.com/dlclark/regexp2"
)

type Iterator interface {
	Next() (Token, error)
	State() interface{}
	SetState(state interface{})
}

type iterator struct {
	// state stores the state of this Lexer at the current point in the lexing.
	// Element 0 is the state of this lexer, and 1 and above are the state of
	// sublexers, if any, that are processing a subset of the text. LexerState[1]
	// is the state for the lexer at depth 1, [2] for depth 2, and so on.
	state     LexerState
	sublexers []*iterator
	rules     Rules
	depth     int
}

func newIterator(text []rune, rules Rules) *iterator {
	iter := &iterator{
		state: LexerState{stack: NewStack(), text: text},
		rules: rules,
	}

	return iter
}

// replace \r and \r\n with \n
// same as strings.ReplaceAll but more efficient
func ensureLF(text []rune) ([]rune, offsetMap) {
	var m offsetMap

	var result []rune
	var j int

	appendToResult := func(c rune) {
		if result == nil {
			result = make([]rune, len(text))
		}
		result[j] = c
		j++
	}

	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == '\r' {
			if i < len(text)-1 && text[i+1] == '\n' {
				m.push(i)
				continue
			}
			c = '\n'
		}
		appendToResult(c)
	}

	if result == nil {
		// No \r found
		return text, m
	}

	return result[:j], m
}

func (i *iterator) pushState(state string) error {
	s, ok := i.rules.Get(state)
	if !ok {
		return fmt.Errorf("No state %s", state)
	}
	i.state.stack.Push(s)
	return nil
}

func (i *iterator) Next() (Token, error) {
	i.pushRootStateIfNeeded()

	switch i.state.stage {
	case stageReadyToMatch:
		return i.nextInReadyToMatchStage()
	case stageWithinGroups:
		return i.nextInWithinGroupsStage()
	case stageRunningSublexer:
		return i.nextInSublexer()
	default:
		return Token{}, fmt.Errorf("Unsupported lexer stage %d", i.state.stage)
	}
}

func (i *iterator) nextInReadyToMatchStage() (tok Token, err error) {
	if i.state.index >= len(i.state.text) {
		debugf("iterator.nextInReadyToMatchStage(%d): Current index %d is past the end of the text. Text has length %d. Returning EOFType",
			i.depth, i.state.index, len(i.state.text))
		return Token{Type: EOFType, Value: nil}, nil
	}

	state := i.state.stack.Top()
	debugf("iterator.nextInReadyToMatchStage(%d): Matching a full rule in top state %s", i.depth, state.name)
	match, rule := state.match(i.state.text[i.state.index:])
	if match == nil {
		debugf("iterator.nextInReadyToMatchStage(%d): No rule in the rule sequence matched", i.depth)
		return Token{Type: Error, Value: nil}, nil
	}

	// TODO: carriage returns?

	if rule.byGroups != nil {
		i.prepareToIterateGroups(rule, match)
		return i.Next()
	}

	if rule.tok == 0 {
		debugf("iterator.nextInReadyToMatchStage(%d): rule provides no token\n", i.depth)
		tok = Token{}
	} else {
		debugf("iterator.nextInReadyToMatchStage(%d): will return token for entire match\n", i.depth)
		// Use entire match
		tok = i.tokenOfEntireMatch(rule.tok, match)
		g := match.GroupByNumber(0)
		debugf("iterator.nextInReadyToMatchStage(%d): Moving index from %d to %d (some text there is: '%s')", i.depth, i.state.index, i.state.index+g.Length,
			aLittleText(i.state.text, i.state.index+g.Length))
		i.state.index += g.Length
	}
	i.handleRuleState(rule)

	if rule.tok == 0 {
		debugf("iterator.nextInReadyToMatchStage(%d): recursing to generate token\n", i.depth)
		return i.Next()
	}

	debugf("iterator.nextInReadyToMatchStage(%d): returning token %s", i.depth, tok)
	return
}

func (i *iterator) prepareToIterateGroups(matchingRule *Rule, match *regexp2.Match) {
	i.state.rule = matchingRule
	i.setCapturesFromMatch(match)
	i.state.groupIndex = 0
	i.state.stage = stageWithinGroups
	i.state.byGroups = matchingRule.byGroups
}

func (it *iterator) setCapturesFromMatch(match *regexp2.Match) {
	it.state.groups = make([]capture, match.GroupCount())
	for i, g := range match.Groups() {
		debugf("iterator.setCapturesFromMatch(%d): group %d in match is at %d of length %d", it.depth, i, g.Index, g.Length)
		it.state.groups[i].start = g.Index
		it.state.groups[i].length = g.Length
	}
}

func (it *iterator) nextInWithinGroupsStage() (tok Token, err error) {
	debugf("iterator.nextInWithinGroupsStage(%d): Will return the next group with index %d (%d/%d)", it.depth, it.state.groupIndex, it.state.groupIndex+1, len(it.state.byGroups))

	byGroup := it.state.byGroups[it.state.groupIndex]
	capture := it.state.groups[it.state.groupIndex+1]

	text := it.state.text[it.state.index:]
	groupText := text[capture.start:capture.end()]
	if byGroup.IsUseSelf() {
		debugf("Lexer.nextInWithinGroupsStage(%d): bygroups %d is a use-self. Creating sub lexer\n", it.depth, it.state.groupIndex)
		it.prepareToUseSublexer(groupText, &capture, &byGroup)
		return it.Next()
	}

	start, end := it.boundsOfGroup(capture.start, capture.length)
	debugf("iterator.nextInWithinGroupsStage(%d): bygroups %d: returning token\n", it.depth, it.state.groupIndex)
	tok = Token{Type: byGroup.tok, Value: groupText, Start: start, End: end}

	it.state.groupIndex++

	if it.state.groupIndex >= len(it.state.byGroups) {
		debugf("iterator.nextInWithinGroupsStage(%d): reached end of the groups, will switch to full match stage\n", it.depth)
		it.handleRuleState(it.state.rule)
		it.state.stage = stageReadyToMatch
		it.state.index += it.state.groups[0].length // Move past the length of the match
		it.clearGroupIterationInfo()
	}

	return tok, nil
}

func (it *iterator) prepareToUseSublexer(groupText []rune, capture *capture, byGroup *byGroupElement) {
	lex := newIterator(groupText, it.rules)
	lex.setOffset(it.state.index + capture.start)
	lex.depth = it.depth + 1
	lex.pushState(byGroup.useSelfState)
	it.state.stage = stageRunningSublexer
	it.sublexers = append(it.sublexers, lex)
}

func (it *iterator) completeGroupIteration() {
	it.handleRuleState(it.state.rule)
	it.state.stage = stageReadyToMatch
	it.state.index += it.state.groups[0].length // Move past the length of the match
	it.clearGroupIterationInfo()
}

func (it *iterator) clearGroupIterationInfo() {
	it.state.groups = it.state.groups[:0]
	it.state.groupIndex = 0
	it.state.byGroups = nil
	it.state.rule = nil
}

func (it *iterator) nextInSublexer() (tok Token, err error) {
	tok, err = it.sublexers[len(it.sublexers)-1].Next()
	if err != nil {
		// On an error, we need to clean up all sublexers.
		// Each parent lexer will clean up it's first child lexer.
		it.sublexers = it.sublexers[:len(it.sublexers)-1]
		return
	}

	if tok.Type == EOFType {
		debugf("iterator.nextInSublexer(%d): sublexer completed\n", it.depth)
		// Sublexer completed. Are we still in gruops?
		it.sublexers = it.sublexers[:len(it.sublexers)-1]
		it.state.stage = stageWithinGroups
		it.state.groupIndex++
		debugf("iterator.nextInSublexer(%d): Setting groupindex to %d (%d/%d)\n", it.depth, it.state.groupIndex, it.state.groupIndex+1, len(it.state.byGroups))
		if it.state.groupIndex > len(it.state.byGroups) {
			debugf("iterator.nextInSublexer(%d): Reached end of groups, will switch to full match stage\n", it.depth)
			it.completeGroupIteration()
		}
		return it.Next()
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

func (i *iterator) pushRootStateIfNeeded() {
	if i.state.stack.Len() == 0 {
		debugf("iterator.pushRootStateIfNeeded(%d): Pushing root state", i.depth)

		s, ok := i.rules.Get("root")
		if ok {
			i.state.stack.Push(s)
		} else {
			debugf("No root state found in lexer")
		}
	}
}

func (it *iterator) tokenOfEntireMatch(typ TokenType, match *regexp2.Match) Token {
	s, e := it.boundsOfCapture(&match.Group.Capture)
	return Token{Type: typ, Value: it.groupText(match.GroupByNumber(0)), Start: s, End: e}
}

func (it *iterator) boundsOfCapture(match *regexp2.Capture) (start, end int) {
	return it.state.index + it.state.offset + match.Index,
		it.state.index + it.state.offset + match.Index + match.Length
}

func (it *iterator) boundsOfGroup(index, length int) (start, end int) {
	return it.state.index + it.state.offset + index,
		it.state.index + it.state.offset + index + length
}

func (it *iterator) handleRuleState(rule *Rule) {
	if rule.popDepth == 0 && rule.pushState == "" {
		return
	}

	if rule.popDepth > 0 {
		debugf("iterator.handleRuleState(%d): Popping %d states", it.depth, rule.popDepth)
		it.state.stack.Pop(rule.popDepth)
		return
	}

	s, ok := it.rules.rules[rule.pushState]
	if !ok {
		msg := fmt.Sprintf("syn.iterator: a rule refers to a state %s that doesn't exist", rule.pushState)
		panic(msg)
	}
	debugf("iterator.handleRuleState(%d): pushing state %s", it.depth, rule.pushState)
	it.state.stack.Push(s)
}

func (it *iterator) groupText(g *regexp2.Group) []rune {
	text := it.state.text[it.state.index:]
	return text[g.Index : g.Index+g.Length]
}

// State returns a representation of the state of the iterator. If the result of State() is saved
// and the iterator is advanced, the iterator can be returned to the same state as when State() was called
// by calling SetState() with the result of State().
//
// The state is invalidated if the text that the Iterator is iterating is changed.
func (it *iterator) State() interface{} {
	states := make([]LexerState, len(it.sublexers)+1)
	states[0] = it.state
	states[0].stack = it.state.stack.Clone()
	for i, sl := range it.sublexers {
		states[i+1] = sl.state
		states[i+1].stack = sl.state.stack.Clone()
	}

	return states
}

func (it *iterator) SetState(s interface{}) {

	state := s.([]LexerState)

	// The lexers and sublexers all use the same rules because the only way to make
	// a sublexer is through usingself (for now).
	// So we can just clone the rules from the base lexer to all the sublexers.
	it.state = state[0]

	it.sublexers = make([]*iterator, len(state)-1)
	for i, state := range state[1:] {
		text := state.text

		it.sublexers[i] = newIterator(text, it.rules)
		it.depth = i + 1
		it.sublexers[i].state = state
	}
}

// setOffset sets a number that is added to the Start and End of each Token
// Next() produces before it is returned.
func (it *iterator) setOffset(i int) {
	it.state.offset = i
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
	offsetIter offsetIterator
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