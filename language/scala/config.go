package scala

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// scalaConfig represents the config extension for the scala language.
type scalaConfig struct {
	// config is the parent gazelle config.
	config *config.Config
}

// GetscalaConfig returns the associated package config.
func getScalaConfig(config *config.Config) *scalaConfig {
	if cfg, ok := config.Exts[ScalaLangName].(*scalaConfig); ok {
		return cfg
	}
	return nil
}

// newScalaConfig initializes a new scalaConfig.
func newScalaConfig(config *config.Config) *scalaConfig {
	return &scalaConfig{
		config: config,
	}
}

// getOrCreateScalaConfig either inserts a new config into the map under the
// language name or replaces it with a clone.
func getOrCreateScalaConfig(config *config.Config) *scalaConfig {
	var cfg *scalaConfig
	if existingExt, ok := config.Exts[ScalaLangName]; ok {
		cfg = existingExt.(*scalaConfig).Clone()
	} else {
		cfg = newScalaConfig(config)
	}
	config.Exts[ScalaLangName] = cfg
	return cfg
}

// Clone copies this config to a new one.
func (c *scalaConfig) Clone() *scalaConfig {
	clone := newScalaConfig(c.config)
	return clone
}

// ParseDirectives is called in each directory visited by gazelle.  The relative
// directory name is given by 'rel' and the list of directives in the BUILD file
// are specified by 'directives'.
func (c *scalaConfig) ParseDirectives(rel string, directives []rule.Directive) (err error) {
	for _, d := range directives {
		switch d.Key {
		// case SomeDirective:
		// 	err = c.parseSomeDirective(d)
		// if err != nil {
		// 	return fmt.Errorf("parse %v: %w", d, err)
		}
	}
	return
}
