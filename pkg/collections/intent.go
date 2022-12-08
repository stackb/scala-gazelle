package collections

import "strings"

// Intent represents an action for an attribute name or "value" optionally
// prefixed by a '+' or '-'.  If the prefix is missing, the Intent is not
// negative.
type Intent struct {
	Value string
	Want  bool
}

func ParseIntent(value string) *Intent {
	value = strings.TrimSpace(value)
	negative := strings.HasPrefix(value, "-")
	positive := strings.HasPrefix(value, "+")
	if negative || positive {
		value = value[1:]
	}
	return &Intent{Value: value, Want: !negative}
}
