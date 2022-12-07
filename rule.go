package syn

import (
	"fmt"
	"github.com/dlclark/regexp2"
)

type State struct {
	name  string
	rules []Rule
}

func (r State) match(text []rune) (*regexp2.Match, *Rule) {
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

type Rules struct {
	// Map of state names to rules in that state
	rules map[string]State
}

func NewRules() Rules {
	return Rules{rules: make(map[string]State)}
}

func (r *Rules) AddState(state State) {
	r.rules[state.name] = state
}

func (r *Rules) Get(stateName string) (state State, ok bool) {
	state, ok = r.rules[stateName]
	return
}

func (r Rules) Validate() error {
	var missing []string

	// Make sure any state that is referred to from a rule actually exists.
	for _, state := range r.rules {
		for _, rule := range state.rules {
			if _, ok := r.rules[rule.pushState]; !ok {
				missing = append(missing, rule.pushState)
			}
		}
	}

	err := r.makeMissingError(missing)
	if err != nil {
		return err
	}

	// Make sure the root state exists
	if _, ok := r.rules["root"]; !ok {
		return fmt.Errorf("The root state is not defined")
	}

	return nil
}

func (r Rules) makeMissingError(missing []string) error {
	if missing == nil || len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("The following states are referred to from rules, but aren't defined: %v\n",
		missing)
}

type Rule struct {
	pattern   *regexp2.Regexp
	tok       TokenType
	pushState string
	popDepth  int
	byGroups  []byGroupElement
	include   string
}

// Match attempts to match the rule. If it succeeds it returns a slice
// holding the index pairs identifying the
// leftmost match of the regular expression in b and the matches, if any, of
// its subexpressions like regexp.FindSubmatchIndex.
// Returns nil if there is no match.
func (r Rule) match(text []rune) (*regexp2.Match, error) {
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

func (b byGroupElement) IsUseSelf() bool {
	return b.useSelfState != ""
}
