package mergeindex

// Config holds configration for the mergeindex tool
type Config struct {
	// OutputFile is the name of the file to write
	OutputFile string
	// PredefinedLabels is a list of bazel labels.  A predefined label is one
	// that should be suppressed from label resolution.
	PredefinedLabels string
	// PreferredLabels is a mapping of string->string.
	PreferredLabels string
}
