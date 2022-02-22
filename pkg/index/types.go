package index

// IndexSpec describes the a list of JarSpecs.
type IndexSpec struct {
	JarSpecs []JarSpec `json:"jarSpecs,omitempty"`
}

// JarSpec describes the symbols provided by a bazel label that produces a jar
// file.
type JarSpec struct {
	// Label is the bazel label that provides the jar
	Label string `json:"label,omitempty"`
	// Filename is the jar filename
	Filename string `json:"filename,omitempty"`
	// Classes is a list of FQNs in the jar
	Classes []string `json:"classes,omitempty"`
	// Packages is a list of packages represented in the jar
	Packages []string `json:"packages,omitempty"`
}

// ScalaFileSpec describes the symbols provided/required by a single source
// file.
type ScalaFileSpec struct {
	// Filename is the source filename
	Filename string `json:"filename,omitempty"`
	// MD5 is the sha256 hash of the file contents
	Sha256 string `json:"sha256,omitempty"`
	// Imports is a list of required imports.
	Imports []string `json:"imports,omitempty"`
	// Packages is a list of provided top-level classes.
	Packages []string `json:"packages,omitempty"`
	// Classes is a list of provided top-level classes.
	Classes []string `json:"classes,omitempty"`
	// Objects is a list of provided top-level classes.
	Objects []string `json:"objects,omitempty"`
	// Traits is a list of provided top-level classes.
	Traits []string `json:"traits,omitempty"`
}

// ScalaRuleSpec represents a list of ScalaFileSpec.
type ScalaRuleSpec struct {
	// Label is the bazel label that names the source file in its srcs list.
	Label string `json:"label,omitempty"`
	// Files is the list of files in the rule
	Srcs []ScalaFileSpec `json:"srcs,omitempty"`
}

// ScalaRuleIndexSpec represents a list of ScalaRuleSpec.
type ScalaRuleIndexSpec struct {
	// Files is the list of files in the rule
	Rules []ScalaRuleSpec `json:"rules,omitempty"`
}
