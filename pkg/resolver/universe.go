package resolver

// Universe is a mashup of interfaces that represents all known symbols, rules,
// etc.
type Universe interface {
	SymbolProviderRegistry
	KnownRuleRegistry
	ConflictResolverRegistry
	Scope
	SymbolResolver
}
