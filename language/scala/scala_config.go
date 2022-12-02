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

	"github.com/stackb/scala-gazelle/pkg/crossresolve"
)

const (
	// ruleDirective is the directive for toggling rule generation.
	ruleDirective = "scala_rule"
	// overrideDirective is the well-know gazelle:override directive for
	// disambiguation overrides.
	overrideDirective = "override"
	// implicitImportDirective adds additional imports for resolution
	implicitImportDirective = "implicit_import"
	// scala_explain_deps prints the reason why deps are included.
	scalaExplainDeps = "scala_explain_deps"
	// scala_expand_srcs replaces the "srcs" attribute with actual srcs, if enabled.
	scalaExplainSrcs = "scala_explain_srcs"
	// mapKindImportNameDirective allows renaming of resolved labels.
	mapKindImportNameDirective = "map_kind_import_name"
)

// scalaConfig represents the config extension for the a scala package.
type scalaConfig struct {
	// ruleIndex is the global rule map.
	ruleIndex crossresolve.RuleIndex
	// config is the parent gazelle config.
	config *config.Config
	// rel is the relative directory.
	rel string
	// exclude patterns for rules that should be skipped for this package.
	rules map[string]*RuleConfig
	// overrides patterns are parsed from 'gazelle:override scala glob IMPORT LABEL'
	overrides []*overrideSpec
	// implicitImports are parsed from 'gazelle:implicit_import scala foo bar [baz]...'
	implicitImports []*implicitImportSpec
	// map kinds are parsed from 'gazelle:map_kind_import_name
	mapKindImportNames map[string]mapKindImportNameSpec
	// explainDeps is a flag to print additional comments on deps & exports
	explainDeps bool
	// explainSrcs is a flag to print additional comments on srcs
	explainSrcs bool
}

// newScalaConfig initializes a new scalaConfig.
func newScalaConfig(ruleIndex crossresolve.RuleIndex, config *config.Config, rel string) *scalaConfig {
	return &scalaConfig{
		ruleIndex:          ruleIndex,
		config:             config,
		rel:                rel,
		rules:              make(map[string]*RuleConfig),
		overrides:          make([]*overrideSpec, 0),
		implicitImports:    make([]*implicitImportSpec, 0),
		mapKindImportNames: make(map[string]mapKindImportNameSpec),
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
func getOrCreateScalaConfig(ruleIndex crossresolve.RuleIndex, config *config.Config, rel string) *scalaConfig {
	var cfg *scalaConfig
	if existingExt, ok := config.Exts[ScalaLangName]; ok {
		cfg = existingExt.(*scalaConfig).clone(config, rel)
		cfg.rel = rel
	} else {
		cfg = newScalaConfig(ruleIndex, config, rel)
	}
	config.Exts[ScalaLangName] = cfg
	return cfg
}

// clone copies this config to a new one.
func (c *scalaConfig) clone(config *config.Config, rel string) *scalaConfig {
	clone := newScalaConfig(c.ruleIndex, config, rel)
	clone.explainDeps = c.explainDeps
	clone.explainSrcs = c.explainSrcs
	for k, v := range c.rules {
		clone.rules[k] = v.clone()
	}
	for k, v := range c.mapKindImportNames {
		clone.mapKindImportNames[k] = v
	}
	clone.overrides = c.overrides[:]
	clone.implicitImports = c.implicitImports[:]
	return clone
}

// LookupRule implements the crossresolve.RuleIndex interface.  It also
// translates relative labels into their absolute form.
func (c *scalaConfig) LookupRule(from label.Label) (*rule.Rule, bool) {
	if c.ruleIndex == nil || from.Name == "" {
		return nil, false
	}
	if from.Repo == "" {
		from = label.New(c.config.RepoName, from.Pkg, from.Name)
	}
	if from.Pkg == "" && from.Repo == c.config.RepoName {
		from = label.New(from.Repo, c.rel, from.Name)
	}
	return c.ruleIndex.LookupRule(from)
}

// parseDirectives is called in each directory visited by gazelle.  The relative
// directory name is given by 'rel' and the list of directives in the BUILD file
// are specified by 'directives'.
func (c *scalaConfig) parseDirectives(directives []rule.Directive) (err error) {
	for _, d := range directives {
		switch d.Key {
		case ruleDirective:
			err = c.parseRuleDirective(d)
			if err != nil {
				return fmt.Errorf(`invalid directive: "gazelle:%s %s": %w`, d.Key, d.Value, err)
			}
		case overrideDirective:
			c.parseOverrideDirective(d)
		case implicitImportDirective:
			c.parseImplicitImportDirective(d)
		case scalaExplainDeps:
			c.parseScalaExplainDeps(d)
		case scalaExplainSrcs:
			c.parseScalaExplainSrcs(d)
		case mapKindImportNameDirective:
			c.parseMapKindImportNameDirective(d)
		}
	}
	return
}

func (c *scalaConfig) parseRuleDirective(d rule.Directive) error {
	fields := strings.Fields(d.Value)
	if len(fields) < 3 {
		return fmt.Errorf("expected three or more fields, got %d", len(fields))
	}
	name, param, value := fields[0], fields[1], strings.Join(fields[2:], " ")
	r, err := c.getOrCreateRuleConfig(c.config, name)
	if err != nil {
		return err
	}
	return r.parseDirective(c, name, param, value)
}

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
	c.overrides = append(c.overrides, &o)
}

func (c *scalaConfig) parseImplicitImportDirective(d rule.Directive) {
	parts := strings.Fields(d.Value)
	if len(parts) < 3 {
		log.Printf("invalid gazelle:%s directive: expected 3+ parts, got %d (%v)", implicitImportDirective, len(parts), parts)
		return
	}
	c.implicitImports = append(c.implicitImports, &implicitImportSpec{
		lang: parts[0],
		imp:  parts[1],
		deps: parts[2:],
	})
}

func (c *scalaConfig) parseMapKindImportNameDirective(d rule.Directive) {
	parts := strings.Fields(d.Value)
	if len(parts) != 3 {
		log.Printf("invalid gazelle:%s directive: expected [KIND SRC_NAME DST_NAME], got %v", mapKindImportNameDirective, parts)
		return
	}
	kind := parts[0]
	src := parts[1]
	dst := parts[2]

	c.mapKindImportNames[kind] = mapKindImportNameSpec{src: src, dst: dst}
}

func (c *scalaConfig) parseScalaExplainDeps(d rule.Directive) {
	parts := strings.Fields(d.Value)
	if len(parts) != 1 {
		log.Printf("invalid gazelle:%s directive: expected 1+ parts, got %d (%v)", scalaExplainDeps, len(parts), parts)
		return
	}
	c.explainDeps = parts[0] == "true"
}

func (c *scalaConfig) parseScalaExplainSrcs(d rule.Directive) {
	parts := strings.Fields(d.Value)
	if len(parts) != 1 {
		log.Printf("invalid gazelle:%s directive: expected 1+ parts, got %d (%v)", scalaExplainSrcs, len(parts), parts)
		return
	}
	c.explainSrcs = parts[0] == "true"
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

func (c *scalaConfig) getImplicitImports(lang, imp string) (deps []string) {
	for _, d := range c.implicitImports {
		if d.lang != lang {
			continue
		}
		if d.imp != imp {
			continue
		}
		deps = append(deps, d.deps...)
	}
	return
}

func (c *scalaConfig) getConfiguredRule(name string) (*RuleConfig, bool) {
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

func (c *scalaConfig) Overrides() []*overrideSpec {
	return c.overrides
}

type overrideSpec struct {
	imp  resolve.ImportSpec
	lang string
	dep  label.Label
}

type implicitImportSpec struct {
	// lang is the language to which this implicit applies.  Always 'scala' for now.
	lang string
	// imp is the "source" dependency (e.g. LazyLogging)
	imp string
	// dep is the "destination" dependencies (e.g. org.slf4j.Logger)
	deps []string
}

type mapKindImportNameSpec struct {
	// src is the label name to match
	src string
	// dst is the label name to rewrite
	dst string
}

func (m *mapKindImportNameSpec) Rename(from label.Label) label.Label {
	if !(m.src == from.Name || m.src == "%{name}") {
		return from
	}
	to := label.New(from.Repo, from.Pkg, strings.ReplaceAll(m.dst, "%{name}", from.Name))
	// log.Printf("matched map_kind_import_name", m, from, "->", to)
	return to
}
