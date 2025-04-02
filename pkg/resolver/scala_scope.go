package resolver

import (
	"fmt"
	"strings"
)

type ScalaScope struct {
	chain    []Scope
	parent   Scope
	scala    Scope
	javaLang Scope
	root     Scope
}

func NewScalaScope(parent Scope) (*ScalaScope, error) {
	scala, ok := parent.GetScope("scala")
	if !ok {
		return nil, fmt.Errorf("scala.* package not found (scala builtins will not resolve)")
	}

	javaLang, ok := parent.GetScope("java.lang")
	if !ok {
		return nil, fmt.Errorf("java.lang.* package not found (java builtins will not resolve)")
	}

	root := NewTrimPrefixScope("_root_.", parent)

	return &ScalaScope{
		chain:    []Scope{parent, scala, javaLang, root},
		parent:   parent,
		scala:    scala,
		javaLang: javaLang,
		root:     root,
	}, nil
}

// PutSymbol is not supported and will panic.
func (r *ScalaScope) PutSymbol(known *Symbol) error {
	return fmt.Errorf("unsupported operation: PutSymbol")
}

// GetSymbol implements part of the Scope interface
func (r *ScalaScope) GetSymbol(imp string) (*Symbol, bool) {
	for _, next := range r.chain {
		if known, ok := next.GetSymbol(imp); ok {
			if known.Name == "scala" || known.Name == "java.lang" {
				continue
			}
			return known, true
		}
	}
	return nil, false
}

// GetScope implements part of the resolver.Scope interface.
func (r *ScalaScope) GetScope(imp string) (Scope, bool) {
	for _, next := range r.chain {
		if scope, ok := next.GetScope(imp); ok {
			if scope == r.scala || scope == r.javaLang {
				continue
			}
			return scope, true
		}
	}
	return nil, false
}

// GetSymbols implements part of the Scope interface
func (r *ScalaScope) GetSymbols(prefix string) []*Symbol {
	for _, next := range r.chain {
		if known := next.GetSymbols(prefix); len(known) > 0 {
			return known
		}
	}
	return nil
}

// String implements the fmt.Stringer interface
func (r *ScalaScope) String() string {
	var buf strings.Builder
	for i, next := range r.chain {
		buf.WriteString(fmt.Sprintf("--- layer %d ---\n", i))
		buf.WriteString(next.String())
		buf.WriteRune('\n')
	}
	return buf.String()
}
