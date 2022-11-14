package syn

import "fmt"

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

func (l *Lexer) Next() Token {
	if l.state.index >= len(l.text) {
		return Token{Typ: EOF, Value: nil}
	}

	rules := l.state.stack.Top()
	match, rule := rules.match(l.text[l.state.index:])
	if match == nil {
		return Token{Error, nil}
	}

	// Got a match. Now emit the token, and change state if needed.
	t := Token{Typ: rule.tok, Value: l.matchText(match)}
	l.state.index = match[1]	
	
	l.handleRuleState(rule)

	return t
}

func (l *Lexer) handleRuleState(rule Rule) {
	if rule.newState == "" {
		return
	}

	if rule.mustPop() {
		l.state.stack.Pop()
		return
	}

	s, ok := l.rules.rules[rule.newState]
	if !ok {
		msg := fmt.Sprintf("syn.Lexer: a rule refers to a state %s that doesn't exist", rule.newState)
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

func (s Stack) Pop() (list RuleSequence) {
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


