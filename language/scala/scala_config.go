package scala

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	// ruleDirective is the directive for toggling rule generation.
	ruleDirective = "scala_rule"
	// overrideDirective is the directive for disambiguation overrides.
	overrideDirective = "override"
)

// scalaConfig represents the config extension for the a scala package.
type scalaConfig struct {
	// config is the parent gazelle config.
	config *config.Config
	// exclude patterns for rules that should be skipped for this package.
	rules map[string]*RuleConfig
	// overrides patterns are parsed from 'gazelle:override scala glob IMPORT LABEL'
	overrides []*overrideSpec
}

// newScalaConfig initializes a new scalaConfig.
func newScalaConfig(config *config.Config) *scalaConfig {
	return &scalaConfig{
		config:    config,
		rules:     make(map[string]*RuleConfig),
		overrides: make([]*overrideSpec, 0),
	}
}

// getScalaConfig returns the scala config.  Can be nil.
func getScalaConfig(config *config.Config) *scalaConfig {
	if existingExt, ok := config.Exts[ScalaLangName]; ok {
		return existingExt.(*scalaConfig)
	} else {
		return nil
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
	for k, v := range c.rules {
		clone.rules[k] = v.clone()
	}
	clone.overrides = c.overrides[:]
	return clone
}

// ParseDirectives is called in each directory visited by gazelle.  The relative
// directory name is given by 'rel' and the list of directives in the BUILD file
// are specified by 'directives'.
func (c *scalaConfig) ParseDirectives(rel string, directives []rule.Directive) (err error) {
	for _, d := range directives {
		switch d.Key {
		case ruleDirective:
			err = c.parseRuleDirective(d)
			if err != nil {
				return fmt.Errorf("parse %v: %w", d, err)
			}
		case overrideDirective:
			c.parseOverrideDirective(d)
		}
	}
	return
}

func (c *scalaConfig) parseRuleDirective(d rule.Directive) error {
	fields := strings.Fields(d.Value)
	if len(fields) < 3 {
		return fmt.Errorf("invalid directive %v: expected three or more fields, got %d", d, len(fields))
	}
	name, param, value := fields[0], fields[1], strings.Join(fields[2:], " ")
	r, err := c.getOrCreateRuleConfig(c.config, name)
	if err != nil {
		return fmt.Errorf("invalid proto_rule directive %+v: %w", d, err)
	}
	return r.parseDirective(c, name, param, value)
}

// parseOverrideDirective parses a resolve directive.  We only consider the
// 4-part resolve form here, and only those for 'scala' language.
func (c *scalaConfig) parseOverrideDirective(d rule.Directive) {
	parts := strings.Fields(d.Value)
	o := overrideSpec{}
	var lbl string
	if len(parts) != 4 {
		return
	}
	if parts[0] != ScalaLangName {
		return
	}
	if parts[1] != "glob" {
		return
	}

	o.imp.Lang = parts[0]
	o.lang = parts[1]
	o.imp.Imp = parts[2]
	lbl = parts[3]

	var err error
	o.dep, err = label.Parse(lbl)
	if err != nil {
		log.Printf("gazelle:override %s: %v", d.Value, err)
		return
	}
	// o.dep = o.dep.Abs("", c.rel) // TODO(pcj): this is really needed?
	c.overrides = append(c.overrides, &o)
}

func (c *scalaConfig) getOrCreateRuleConfig(config *config.Config, name string) (*RuleConfig, error) {
	r, ok := c.rules[name]
	if !ok {
		r = NewRuleConfig(config, name)
		r.Implementation = name
		c.rules[name] = r
	}
	return r, nil
}

func (c *scalaConfig) GetConfiguredRule(name string) (*RuleConfig, bool) {
	rc, ok := c.rules[name]
	return rc, ok
}

// configuredRules returns a determinstic ordered list of configured
// rules
func (c *scalaConfig) configuredRules() []*RuleConfig {
	names := make([]string, 0)
	for name := range c.rules {
		names = append(names, name)
	}
	sort.Strings(names)
	rules := make([]*RuleConfig, 0)
	for _, name := range names {
		rules = append(rules, c.rules[name])
	}
	return rules
}

// DeduplicateAndSort removes duplicate entries and sorts the list
func DeduplicateAndSort(in []string) (out []string) {
	seen := make(map[string]bool)
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return
}

type overrideSpec struct {
	imp  resolve.ImportSpec
	lang string
	dep  label.Label
}
