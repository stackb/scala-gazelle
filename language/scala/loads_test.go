package scala

import "fmt"

func ExampleLanguage_Loads() {
	lang := NewLanguage()
	for _, info := range lang.Loads() {
		fmt.Printf("%+v\n", info)
	}
	// output:
	// {Name://custom/scala:scala.bzl Symbols:[scala_binary] After:[]}
	// {Name:@io_bazel_rules_scala//scala:scala.bzl Symbols:[scala_binary scala_library scala_library scala_macro_library scala_test] After:[]}
}
