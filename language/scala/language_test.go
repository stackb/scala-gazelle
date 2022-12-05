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
	// resolve_glob
	// resolve_with
	// scala_explain_deps
	// scala_annotate_imports
	// resolve_kind_rewrite_name
}
