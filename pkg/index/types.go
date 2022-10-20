package index

// IndexSpec describes the a list of JarSpecs.
type IndexSpec struct {
	// JarSpecs is the list of jars in the index.
	JarSpecs []*JarSpec `json:"jarSpecs,omitempty"`
	// Predefined is a list of labels that do not need to be explicity provided
	// in deps.  Examples would include platform jar class (e.g. the jar that
	// contains java.lang.Object) and the scala stdlib.
	Predefined []string `json:"predefined,omitempty"`
	// Preferred is a list of labels that should be used in the case of a
	// resolve ambiguity.
	Preferred []string `json:"preferred,omitempty"`
}

// JarSpec describes the symbols provided by a bazel label that produces a jar
// file.
type JarSpec struct {
	Symbols []string         `json:"symbols,omitempty"`
	Files   []*ClassFileSpec `json:"files,omitempty"`
	// Label is the bazel label that provides the jar
	Label string `json:"label,omitempty"`
	// Filename is the jar filename
	Filename string `json:"filename,omitempty"`
	// Classes is a list of FQNs in the jar
	Classes []string `json:"classes,omitempty"`
	// Packages is a list of packages represented in the jar
	Packages []string `json:"packages,omitempty"`
	// Extends is a mapping from class to symbol that it extends
	Extends map[string]string `json:"extends,omitempty"`
}

// // JarSpec describes the symbols provided by a bazel label that produces a jar
// // file.
// type JarSpecOld struct {
// 	Symbols []string `json:"symbols,omitempty"`
// 	Files   []*ClassFileSpec
// 	// Label is the bazel label that provides the jar
// 	Label string `json:"label,omitempty"`
// 	// Filename is the jar filename
// 	Filename string `json:"filename,omitempty"`
// 	// Classes is a list of FQNs in the jar
// 	Classes []string `json:"classes,omitempty"`
// 	// Packages is a list of packages represented in the jar
// 	Packages []string `json:"packages,omitempty"`
// 	// Extends is a mapping from class to symbol that it extends
// 	Extends map[string]string `json:"extends,omitempty"`
// }

type ClassFileSpec struct {
	// Name is the class FQN
	Name string `json:"name"`
	// Classes is the list of classes in the constant pool.
	Classes      []int             `json:"classes,omitempty"`
	Symbols      []string          `json:"symbols,omitempty"`
	Superclasses []string          `json:"superclasses,omitempty"`
	Interfaces   []string          `json:"interfaces,omitempty"`
	Fields       []ClassFieldSpec  `json:"fields,omitempty"`
	Methods      []ClassMethodSpec `json:"methods,omitempty"`
}

type ClassFieldSpec struct {
	// Name of the field
	Name string   `json:"name"`
	Type TypeSpec `json:"type"`
}

type TypeSpec struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type TypeParamSpec struct {
	Name string `json:"name"`
}

type ClassMethodSpec struct {
	Name    string                 `json:"name"`
	Returns TypeSpec               `json:"returns"`
	Params  []ClassMethodParamSpec `json:"params"`
	Types   TypeParamSpec          `json:"types"`
	Throws  []TypeSpec             `json:"throwz"`
}

type ClassMethodParamSpec struct {
	Returns TypeSpec `json:"type"`
}

// ScalaFileSpec describes the symbols provided/required by a single source
// file.
type ScalaFileSpec struct {
	// Filename is the source filename
	Filename string `json:"filename,omitempty"`
	// Sha256 is the sha256 hash of the file contents
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
	// Types is a list of provided top-level types (in package objects).
	Types []string `json:"types,omitempty"`
	// Vals is a list of provided top-level vals (in package objects).
	Vals []string `json:"vals,omitempty"`
	// Names is a list of simple function calls.  In practice these look like
	// constructor invocations.
	Names []string `json:"names,omitempty"`
	// Extends is a mapping from the base type to a list of symbol names.
	Extends map[string][]string `json:"extends,omitempty"`
}

// ScalaRuleSpec represents a list of ScalaFileSpec.
type ScalaRuleSpec struct {
	// Label is the bazel label that names the source file in its srcs list.
	Label string `json:"label,omitempty"`
	// Kind is the kind of rule named by Label.
	Kind string `json:"kind,omitempty"`
	// Files is the list of files in the rule
	Srcs []*ScalaFileSpec `json:"srcs,omitempty"`
}

// ScalaRuleIndexSpec represents a list of ScalaRuleSpec.
type ScalaRuleIndexSpec struct {
	// Rules is the list of rule specs in the index.
	Rules []*ScalaRuleSpec `json:"rules,omitempty"`
}

// ScalaCompileSpec describes the symbols derived from attempting to compile a scala source file.
type ScalaCompileSpec struct {
	// NotFound is a list of types that were not found (e.g. "not found: value DateUtils")
	NotFound []*NotFoundSymbol `json:"notFound,omitempty"`
	// E.g. "object Session is not a member of package com.foo.core"
	NotMember []*NotMemberSymbol `json:"notMember,omitempty"`
}

type NotFoundSymbol struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type NotMemberSymbol struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Package string `json:"package"`
}
