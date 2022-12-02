package syn

import (
	"fmt"
	"regexp"
)

type RuleSequence []Rule

func (r RuleSequence) match(text []byte) ([]int, *Rule) {
	for i, rule := range r {
		res := rule.match(text)
		if res != nil {
			return res, &r[i]
		}
	}
	return nil, nil
}

type Rules struct {
	// Map of state names to rules in that state
	rules map[string]RuleSequence
}

func NewRules() Rules {
	return Rules{rules: make(map[string]RuleSequence)}
}

func (r *Rules) AddRuleSequence(state string, seq RuleSequence) {
	if seq == nil {
		return
	}

	r.rules[state] = seq
}

func (r *Rules) RuleSequenceForState(state string) RuleSequence {
	s, ok := r.rules[state]
	if !ok {
		return nil
	}
	return s
}

func (r Rules) Validate() error {
	var missing []string

	// Make sure any state that is referred to from a rule actually exists.
	for _, rules := range r.rules {
		for _, rule := range rules {
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
	pattern   *regexp.Regexp
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
func (r Rule) match(text []byte) []int {
	return r.pattern.FindSubmatchIndex(text)
}

type byGroupElement struct {
	tok          TokenType
	useSelfState string
}

func (b byGroupElement) IsUseSelf() bool {
	return b.useSelfState != ""
}
