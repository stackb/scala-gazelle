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
	// resolve_conflicts
	// resolve_file_symbol_name
	// resolve_glob
	// resolve_kind_rewrite_name
	// resolve_with
	// scala_annotate
	// scala_rule
}
