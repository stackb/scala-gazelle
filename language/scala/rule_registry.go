package scala

import (
	"errors"
	"sort"
)

// ErrUnknownRule is the error returned when a rule is not known.
var ErrUnknownRule = errors.New("unknown rule")

// RuleRegistry represents a library of rule implementations.
type RuleRegistry interface {
	// RuleNames returns a sorted list of rule names.
	RuleNames() []string
	// LookupRule returns the implementation under the given name.  If the rule
	// is not found, ErrUnknownRule is returned.
	LookupRule(name string) (RuleInfo, error)
	// MustRegisterRule installs a RuleInfo implementation under the given
	// name in the global rule registry.  Panic will occur if the same rule is
	// registered multiple times.
	MustRegisterRule(name string, rule RuleInfo) RuleRegistry
}

// Rules returns a reference to the global RuleRegistry
func Rules() RuleRegistry {
	return globalRuleRegistry
}

// registry is the default registry singleton.
var globalRuleRegistry = &registry{
	rules: make(map[string]RuleInfo),
}

// registry implements RuleRegistry.
type registry struct {
	rules map[string]RuleInfo
}

// RuleNames implements part of the RuleRegistry interface.
func (p *registry) RuleNames() []string {
	names := make([]string, 0)
	for name := range p.rules {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// MustRegisterRule implements part of the RuleRegistry interface.
func (p *registry) MustRegisterRule(name string, rule RuleInfo) RuleRegistry {
	_, ok := p.rules[name]
	if ok {
		panic("duplicate scala_rule registration: " + name)
	}
	p.rules[name] = rule
	return p
}

// LookupRule implements part of the RuleRegistry interface.
func (p *registry) LookupRule(name string) (RuleInfo, error) {
	rule, ok := p.rules[name]
	if !ok {
		return nil, ErrUnknownRule
	}
	return rule, nil
}
