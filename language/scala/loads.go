package scala

import (
	"log"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Loads implements part of the language.Language interface
func (sl *scalaLang) Loads() []rule.LoadInfo {
	// Merge symbols
	symbolsByLoadName := make(map[string][]string)

	for _, name := range sl.ruleProviderRegistry.ProviderNames() {
		provider, ok := sl.ruleProviderRegistry.LookupProvider(name)
		if !ok {
			log.Fatalf("unknown rule provider: %q", name)
		}
		load := provider.LoadInfo()
		symbolsByLoadName[load.Name] = append(symbolsByLoadName[load.Name], load.Symbols...)
	}

	// Ensure names are sorted otherwise order of load statements can be
	// non-deterministic
	keys := make([]string, 0)
	for name := range symbolsByLoadName {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	// Build final load list
	loads := make([]rule.LoadInfo, 0)
	for _, name := range keys {
		symbols := symbolsByLoadName[name]
		sort.Strings(symbols)
		loads = append(loads, rule.LoadInfo{
			Name:    name,
			Symbols: symbols,
		})
	}
	return loads
}

func fullyQualifiedLoadName(loads []*rule.Load, kind string) string {
	for _, load := range loads {
		for _, pair := range load.SymbolPairs() {
			// when there is no aliasing, pair.From == pair.To, so this covers
			// both cases (aliases and not).
			if pair.From == pair.To && pair.From == kind {
				return load.Name() + "%" + pair.From
			}
			if pair.To == kind {
				return load.Name() + "%" + pair.From
			}
		}
	}
	// no match, just return the kind (e.g. native.java_binary)
	return kind
}
