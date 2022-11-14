package syn

import (
	"fmt"
	"regexp"
)

type RuleSequence []Rule

func (r RuleSequence) match(text []byte) ([]int, Rule) {
	for _, rule := range r {
		res := rule.match(text)
		if res != nil {
			return res, rule
		}
	}
	return nil, Rule{}
}

type Rules struct {
	// Map of state names to rules in that state
	rules map[string]RuleSequence
}

func (r Rules) Validate() error {
	var missing []string

	// Make sure any state that is referred to from a rule actually exists.
	for _, rules := range r.rules {
		for _, rule := range rules {
			if _, ok := r.rules[rule.newState]; !ok {
				missing = append(missing, rule.newState)
			}
		}
	}
	
	err := r.makeMissingError(missing)
	if err != nil {
		return err
	}

	// Make sure the root state exists	
	if _, ok :=  r.rules["root"]; !ok {
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
	pattern  *regexp.Regexp
	tok      TokenType
	newState string
}

// Match attempts to match the rule. If it succeeds it returns a slice
// holding the index pairs identifying the
// leftmost match of the regular expression in b and the matches, if any, of
// its subexpressions like regexp.FindSubmatchIndex.
// Returns nil if there is no match.
func (r Rule) match(text []byte) []int {
	return r.pattern.FindSubmatchIndex(text)
}

func (r Rule) mustPop() bool {
	return r.newState == "#pop"
}

