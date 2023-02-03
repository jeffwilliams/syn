package syn

import (
	"bytes"
	"fmt"

	"github.com/dlclark/regexp2"
)

type Iterator interface {
	Next() (Token, error)
	State() IteratorState
	SetState(state IteratorState)
}

type IteratorState interface {
	Equal(s IteratorState) bool
	// SetIndex sets the index of the next input rune in the text. This can be used
	// to adjust the position stored in the state if an earlier subset of the input
	// text has been inserted or deleted, but the state of the iterator may still
	// apply.
	SetIndex(i int)
	AddToIndex(delta int)
}

type iterator struct {
	// state stores the state of this Lexer at the current point in the lexing.
	// Element 0 is the state of this lexer, and 1 and above are the state of
	// sublexers, if any, that are processing a subset of the text. LexerState[1]
	// is the state for the lexer at depth 1, [2] for depth 2, and so on.
	text      []rune
	state     lexerState
	sublexers []*iterator
	rules     rules
	depth     int
}

func newIterator(text []rune, rulez rules) *iterator {
	iter := &iterator{
		text:  text,
		state: lexerState{stack: newStack()},
		rules: rulez,
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
	if i.state.index >= len(i.text) {
		debugf("iterator.nextInReadyToMatchStage(%d): Current index %d is past the end of the text. Text has length %d. Returning EOFType",
			i.depth, i.state.index, len(i.text))
		return Token{Type: EOFType, Value: nil}, nil
	}

	state := i.state.stack.Top()
	debugf("iterator.nextInReadyToMatchStage(%d): Matching a full rule in top state %s", i.depth, state.name)
	match, rule := state.match(i.text[i.state.index:])
	if match == nil {
		debugf("iterator.nextInReadyToMatchStage(%d): No rule in the rule sequence matched", i.depth)
		i.state.index++
		if i.state.index < len(i.text) && i.text[i.state.index] == '\n' {
			// This idea is taken from Chroma, which also took it from Pygments. To quote:
			//
			// "If the RegexLexer encounters a newline that is flagged as an error token, the stack is
			// emptied and the lexer continues scanning in the 'root' state. This can help producing
			// error-tolerant highlighting for erroneous input, e.g. when a single-line string is not
			// closed."
			//
			// Basically we keep making progress character by character and try to reset.
			// TODO: This could be slow for large files; perhaps this should be an option.
			i.state.stack.Clear()
			i.pushRootStateIfNeeded()
		}
		return Token{Type: Error, Value: nil}, nil
	}

	if rule.byGroups != nil {
		i.prepareToIterateGroups(rule, match)
		return i.Next()
	}

	if rule.IsUseSelf() {
		groupText := i.groupText(match.GroupByNumber(0))
		i.prepareToUseSublexer(rule, groupText, 0, rule.useSelfState)
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
			aLittleText(i.text, i.state.index+g.Length))
		i.state.index += g.Length
	}

	err = i.handleRuleState(rule)
	if err != nil {
		return
	}

	if rule.tok == 0 {
		debugf("iterator.nextInReadyToMatchStage(%d): recursing to generate token\n", i.depth)
		return i.Next()
	}

	debugf("iterator.nextInReadyToMatchStage(%d): returning token %s", i.depth, tok)
	return
}

func (i *iterator) prepareToIterateGroups(matchingRule *rule, match *regexp2.Match) {
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

	text := it.text[it.state.index:]
	groupText := text[capture.start:capture.end()]
	if byGroup.IsUseSelf() {
		debugf("Lexer.nextInWithinGroupsStage(%d): bygroups %d is a use-self. Creating sub lexer\n", it.depth, it.state.groupIndex)
		it.prepareToUseSublexer(it.state.rule, groupText, capture.start, byGroup.useSelfState)
		return it.Next()
	}

	start, end := it.boundsOfGroup(capture.start, capture.length)
	debugf("iterator.nextInWithinGroupsStage(%d): bygroups %d: returning token\n", it.depth, it.state.groupIndex)
	tok = Token{Type: byGroup.tok, Value: groupText, Start: start, End: end}

	it.state.groupIndex++

	if it.state.groupIndex >= len(it.state.byGroups) {
		debugf("iterator.nextInWithinGroupsStage(%d): reached end of the groups, will switch to full match stage\n", it.depth)
		err = it.handleRuleState(it.state.rule)
		if err != nil {
			return
		}
		it.state.stage = stageReadyToMatch
		it.state.index += it.state.groups[0].length // Move past the length of the match
		it.clearGroupIterationInfo()
	}

	return tok, nil
}

func (it *iterator) prepareToUseSublexer(rule *rule, groupText []rune, captureStart int, state string) {
	lex := newIterator(groupText, it.rules)
	lex.setOffset(it.state.index + captureStart)
	lex.depth = it.depth + 1
	lex.pushState(state)
	it.state.stage = stageRunningSublexer
	it.sublexers = append(it.sublexers, lex)
	it.state.rule = rule
}

func (it *iterator) completeGroupIteration() error {
	err := it.handleRuleState(it.state.rule)
	if err != nil {
		return err
	}
	it.state.stage = stageReadyToMatch
	if len(it.state.byGroups) == 0 {
		// byGroups has zero elements when this is a usingself that is not within groups; it's against the complete match of the rule's pattern
		it.state.index += len(it.text)
	} else {
		it.state.index += it.state.groups[0].length // Move past the length of the match
	}
	it.clearGroupIterationInfo()
	return nil
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
		if it.state.groupIndex >= len(it.state.byGroups) {
			debugf("iterator.nextInSublexer(%d): Reached end of groups, will switch to full match stage\n", it.depth)
			err = it.completeGroupIteration()
			if err != nil {
				return
			}
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

func (it *iterator) handleRuleState(rule *rule) error {
	if rule.popDepth == 0 && rule.pushState == "" {
		return nil
	}

	if rule.popDepth > 0 {
		debugf("iterator.handleRuleState(%d): Popping %d states", it.depth, rule.popDepth)
		it.state.stack.Pop(rule.popDepth)
		return nil
	}

	s, ok := it.rules.rules[rule.pushState]
	if !ok {
		return fmt.Errorf("syn.iterator: a rule refers to a state %s that doesn't exist", rule.pushState)
	}
	debugf("iterator.handleRuleState(%d): pushing state %s", it.depth, rule.pushState)
	it.state.stack.Push(s)
	return nil
}

func (it *iterator) groupText(g *regexp2.Group) []rune {
	text := it.text[it.state.index:]
	return text[g.Index : g.Index+g.Length]
}

// State returns a representation of the state of the iterator. If the result of State() is saved
// and the iterator is advanced, the iterator can be returned to the same state as when State() was called
// by calling SetState() with the result of State().
//
// The state is invalidated if the text that the Iterator is iterating is changed.
func (it *iterator) State() IteratorState {
	states := make(lexerStates, len(it.sublexers)+1)
	states[0] = it.state
	states[0].stack = it.state.stack.Clone()
	for i, sl := range it.sublexers {
		states[i+1] = sl.state
		states[i+1].stack = sl.state.stack.Clone()
	}

	return states
}

func (it *iterator) SetState(s IteratorState) {

	state := s.(lexerStates)

	// The lexers and sublexers all use the same rules because the only way to make
	// a sublexer is through usingself (for now).
	// So we can just clone the rules from the base lexer to all the sublexers.
	it.state = state[0]

	it.sublexers = make([]*iterator, len(state)-1)
	for i, state := range state[1:] {
		// TODO: we can't set the text that we're parsing as part of the state
		it.sublexers[i] = newIterator(it.text, it.rules)
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

// lexerState represents the state of the Lexer at some intermediate position in the lexing.
// It determines what token should be matched next based on what has aleady been processed up to
// a certain byte-position in the input text. It can be used to restart lexing from that same point
// in the text.
type lexerState struct {
	stack *stack
	// index is the index of the next input rune to process
	index int
	stage stage
	// offset is the absolute offset of where the beginning of the text parsed by the current lexer
	// represents. It's used when a sublexer is used to lex a subsequence of the text.
	offset int

	groups     []capture        // Groups from the last match. Used when we need to iterate over bygroups. Each entry is a start/end of group.
	groupIndex int              // index for current group we are iterating over
	byGroups   []byGroupElement // The "by groups" items defined in the rule that specify how to handle each group from the match
	rule       *rule            // Rule we are matching the groups for
	offsetIter offsetIterator
}

func (ls lexerState) equal(o *lexerState) bool {
	// We don't compare all fields here, only enough to tell if the
	// lexer would be in the same state in both cases.
	return ls.stacksEqual(o) &&
		ls.index == o.index &&
		ls.stage == o.stage &&
		ls.offset == o.offset &&
		ls.groupsEqual(o) &&
		ls.groupIndex == o.groupIndex &&
		// NOTE: this next line compares the pointers to the rule; fine as long as the rule is created from the same lexer
		ls.rule == o.rule
}

func (ls lexerState) stacksEqual(o *lexerState) bool {
	if ls.stack.Len() != o.stack.Len() {
		return false
	}

	for i, e := range ls.stack.data {
		if e.name != o.stack.data[i].name {
			return false
		}
	}

	return true
}

func (ls lexerState) groupsEqual(o *lexerState) bool {
	if len(ls.groups) != len(o.groups) {
		return false
	}

	for i, e := range ls.groups {
		if !e.equal(&o.groups[i]) {
			return false
		}
	}

	return true
}

func (ls lexerState) String() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "lexerState: \n")
	fmt.Fprintf(&buf, "  index: %d\n", ls.index)
	fmt.Fprintf(&buf, "  stage: %d\n", ls.stage)
	fmt.Fprintf(&buf, "  offset: %d\n", ls.offset)
	fmt.Fprintf(&buf, "  groups: %#v\n", ls.groups)
	fmt.Fprintf(&buf, "  groupIndex: %d\n", ls.groupIndex)
	fmt.Fprintf(&buf, "  rule: %v\n", ls.rule)

	return buf.String()
}

type lexerStates []lexerState

func (ls lexerStates) Equal(o IteratorState) bool {
	other, ok := o.(lexerStates)
	if !ok {
		return false
	}

	for i, e := range ls {
		if !e.equal(&other[i]) {
			return false
		}
	}

	return true
}

func (ls lexerStates) SetIndex(ndx int) {
	for i := range ls {
		ls[i].index = ndx
	}
}

func (ls lexerStates) AddToIndex(ndx int) {
	for i := range ls {
		ls[i].index += ndx
	}
}

func (ls lexerStates) String() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "lexerStates: \n")
	for i, s := range ls {
		fmt.Fprintf(&buf, "  [%d]: %s\n", i, s)
	}

	return buf.String()
}

type capture struct {
	start, length int
}

func (c capture) end() int {
	return c.start + c.length
}

func (c capture) equal(o *capture) bool {
	return c.start == o.start && c.length == o.length
}

type action struct {
	typ       actionType
	tokenType TokenType
}

type actionType int

const (
	pop actionType = iota
	emitToken
)
