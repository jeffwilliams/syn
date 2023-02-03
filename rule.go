package syn

import (
	"bytes"
	"fmt"

	"github.com/dlclark/regexp2"
)

// state represents a state of the lexer. A state consists of a group of rules that are attempted
// to be matched in order. A rule may push a new state onto a logical stack to change the set of rules
// to be used for matching. A rule may also pop Stats off the stack to return to a previous state.
type state struct {
	name  string
	rules []rule
}

func (r state) match(text []rune) (*regexp2.Match, *rule) {
	for i, rule := range r.rules {
		debugf("State.match: for state %s trying rule %d /%s/\n", r.name, i, rule.pattern)
		res, err := rule.match(text)
		if res != nil && err == nil {
			debugf("State.match: rule %d matched\n", i)
			return res, &r.rules[i]
		}
	}
	return nil, nil
}

func (s state) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "State %s:\n", s.name)
	for i, v := range s.rules {
		fmt.Fprintf(&buf, "  rule %d: %s\n", i, v.String())
	}
	return buf.String()

}

// rules is the set of rules in a Lexer
type rules struct {
	// Map of state names to rules in that state
	rules map[string]state
}

// newRules creates an empty Rules
func newRules() rules {
	return rules{rules: make(map[string]state)}
}

// NewRules adds a State to the rules
func (r *rules) AddState(s state) {
	r.rules[s.name] = s
}

// Get retrieves the State with the specified name. If not found, ok is false.
func (r *rules) Get(stateName string) (stat state, ok bool) {
	stat, ok = r.rules[stateName]
	return
}

func (r *rules) Contains(stateName string) (ok bool) {
	_, ok = r.rules[stateName]
	return
}

func (r rules) String() string {
	var buf bytes.Buffer
	for _, v := range r.rules {
		buf.WriteString(v.String())
	}
	return buf.String()
}

// A Rule specifies a regexp to match when lexing at the current position in the text, and an action
// to take if the regexp matches.
type rule struct {
	pattern      *regexp2.Regexp
	tok          TokenType
	pushState    string
	popDepth     int
	byGroups     []byGroupElement
	include      string
	useSelfState string
}

func (r rule) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "(rule /%s/ tok: %s", r.pattern, r.tok)
	if r.pushState != "" {
		fmt.Fprintf(&buf, "  push: %s", r.pushState)
	}
	fmt.Fprintf(&buf, "  pop: %d", r.popDepth)
	if r.byGroups != nil {
		fmt.Fprintf(&buf, "  bygroups size: %d", len(r.byGroups))
	}
	if r.include != "" {
		fmt.Fprintf(&buf, "  include: %s", r.include)
	}
	if r.useSelfState != "" {
		fmt.Fprintf(&buf, "  usingself: %s", r.useSelfState)
	}
	fmt.Fprintf(&buf, ")")
	return buf.String()
}

// IsUseSelf returns true if the Rule specifies that the group should be handled by lexing
// the group text with a new instance of the lexer.
func (r rule) IsUseSelf() bool {
	return r.useSelfState != ""
}

// Match attempts to match the rule. If it succeeds it returns a slice
// holding the index pairs identifying the
// leftmost match of the regular expression in b and the matches, if any, of
// its subexpressions like regexp.FindSubmatchIndex.
// Returns nil if there is no match.
func (r rule) match(text []rune) (*regexp2.Match, error) {
	m, err := r.pattern.FindRunesMatch(text)
	if m != nil && m.Index != 0 {
		return nil, nil
	}
	return m, err
}

type byGroupElement struct {
	tok          TokenType
	useSelfState string
}

// IsUseSelf returns true if the Rule specifies that the group should be handled by lexing
// the group text with a new instance of the lexer.
func (b byGroupElement) IsUseSelf() bool {
	return b.useSelfState != ""
}
