package scala

import (
	"flag"

	"github.com/bazelbuild/bazel-gazelle/config"
)

// RegisterFlags implements part of the language.Language interface
func (sl *scalaLang) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	getOrCreateScalaConfig(c) // ignoring return value, only want side-effect

	for _, r := range sl.resolvers {
		r.RegisterFlags(fs, cmd, c)
	}

	sl.scalaCompiler.RegisterFlags(fs, cmd, c)
	sl.viz.RegisterFlags(fs, cmd, c)

	fs.IntVar(&sl.totalPackageCount, "total_package_count", 0, "number of total packages for the workspace (used for progress estimation)")
}

// CheckFlags implements part of the language.Language interface
func (sl *scalaLang) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	for _, r := range sl.resolvers {
		if err := r.CheckFlags(fs, c); err != nil {
			return err
		}
	}
	if err := sl.scalaCompiler.CheckFlags(fs, c); err != nil {
		return err
	}
	if err := sl.viz.CheckFlags(fs, c); err != nil {
		return err
	}
	return nil
}
