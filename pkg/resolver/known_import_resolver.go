package resolver

// KnownImportResolver is a mashup of interfaces.
type KnownImportResolver interface {
	KnownImportProviderRegistry
	KnownImportRegistry
	KnownRuleRegistry
	ImportResolver
}
