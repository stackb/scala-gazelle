package scala

import "fmt"

func ExampleLanguage_Name() {
	lang := NewLanguage()
	fmt.Println(lang.Name())
	// output:
	// scala
}

func ExampleLanguage_KnownDirectives() {
	lang := NewLanguage()
	for _, d := range lang.KnownDirectives() {
		fmt.Println(d)
	}
	// output:
	// scala_rule
	// override
	// indirect_dependency
	// implicit_import
	// scala_explain_dependencies
	// map_kind_import_name
}
