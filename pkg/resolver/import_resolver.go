package resolver

// ImportResolver is a mashup of interfaces.
type ImportResolver interface {
	KnownImportProviderRegistry
	KnownImportRegistry
	KnownRuleRegistry
	KnownImportResolver
}
