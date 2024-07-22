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
	// scala_debug
	// scala_rule
	// resolve_glob
	// resolve_conflicts
	// resolve_with
	// resolve_file_symbol_name
	// resolve_kind_rewrite_name
}
