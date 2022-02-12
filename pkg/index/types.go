package index

// JarSpec describes the symbols provided by a bazel label that produces a jar
// file.
type JarSpec struct {
	// Label is the bazel label that provides the jar
	Label string `json:"label,omitempty"`
	// Filename is the jar filename
	Filename string `json:"filename,omitempty"`
	// Classes is a list of FQNs in the jar
	Classes []string `json:"classes,omitempty"`
}
