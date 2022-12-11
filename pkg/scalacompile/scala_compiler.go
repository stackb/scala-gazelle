package scalacompile

// ScalaCompiler knows how to compile scala files.  This is not really meant to
// offer "real" compilation of files, but we can use the mostly standard scala
// compiler without a classpath such that a few of the initial passes are run,
// get a bunch of errors back, and use those diagnostics to glean info about the
// type system.
type ScalaCompiler interface {
	// CompileScala compiles the file and returns a compilespec.
	CompileScala(dir string, filenames []string) (*ScalaCompileSpec, error)
}
