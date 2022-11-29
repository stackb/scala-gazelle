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
	// overrideDirective is the well-know gazelle:override directive for
	// disambiguation overrides.
	overrideDirective = "override"
	// indirectDependencyDirective is the directive for declaring indirect
	// dependencies.  For example, if a class explicitly imports
	// 'com.typesafe.scalalogging.LazyLogging', it is also going to need
	// 'org.slf4j.Logger', without mentioning it.
	indirectDependencyDirective = "indirect_dependency"
	// implicitImportDirective adds additional imports for resolution
	implicitImportDirective = "implicit_import"
	// scala_explain_dependencies prints the reason why deps are included.
	scalaExplainDependencies = "scala_explain_dependencies"
	// mapKindImportNameDirective allows renaming of resolved labels.
	mapKindImportNameDirective = "map_kind_import_name"
)

// scalaConfig represents the config extension for the a scala package.
type scalaConfig struct {
	// global is the globalState interface
	global globalState
	// config is the parent gazelle config.
	config *config.Config
	// rel is the relative directory
	rel string
	// exclude patterns for rules that should be skipped for this package.
	rules map[string]*RuleConfig
	// overrides patterns are parsed from 'gazelle:override scala glob IMPORT LABEL'
	overrides []*overrideSpec
	// indirects are parsed from 'gazelle:indirect-dependency scala foo bar'
	indirects []*indirectDependencySpec
	// implicitImports are parsed from 'gazelle:implicit_import scala foo bar [baz]...'
	implicitImports []*implicitImportSpec
	// map kinds are parsed from 'gazelle:map_kind_import_name
	mapKindImportNames map[string]mapKindImportNameSpec
	// explainDependencies is a flag to print additional comments on deps & exports
	explainDependencies bool
}

// newScalaConfig initializes a new scalaConfig.
func newScalaConfig(global globalState, config *config.Config, rel string) *scalaConfig {
	return &scalaConfig{
		global:             global,
		config:             config,
		rel:                rel,
		rules:              make(map[string]*RuleConfig),
		overrides:          make([]*overrideSpec, 0),
		indirects:          make([]*indirectDependencySpec, 0),
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
func getOrCreateScalaConfig(global globalState, config *config.Config, rel string) *scalaConfig {
	var cfg *scalaConfig
	if existingExt, ok := config.Exts[ScalaLangName]; ok {
		cfg = existingExt.(*scalaConfig).clone(config, rel)
		cfg.rel = rel
	} else {
		cfg = newScalaConfig(global, config, rel)
	}
	config.Exts[ScalaLangName] = cfg
	return cfg
}

// clone copies this config to a new one.
func (c *scalaConfig) clone(config *config.Config, rel string) *scalaConfig {
	clone := newScalaConfig(c.global, config, rel)
	clone.explainDependencies = c.explainDependencies
	for k, v := range c.rules {
		clone.rules[k] = v.clone()
	}
	for k, v := range c.mapKindImportNames {
		clone.mapKindImportNames[k] = v
	}
	clone.overrides = c.overrides[:]
	clone.indirects = c.indirects[:]
	clone.implicitImports = c.implicitImports[:]
	return clone
}

func (c *scalaConfig) LookupRule(from label.Label) (*rule.Rule, bool) {
	if c.global == nil {
		return nil, false
	}
	if from.Pkg == "" {
		from = label.New(from.Repo, c.rel, from.Name)
		log.Printf("scalaConfig.LookupRlue from.Pkg assigned to %s", c.rel)
	}
	if from.Repo == "" {
		from = label.New(c.config.RepoName, from.Pkg, from.Name)
		log.Printf("scalaConfig.LookupRlue from.repo assigned to %s", c.config.RepoName)
	}
	if from.Name == "blending" {
		log.Printf("scalaConfig.LookupRlue from %s", from)
	}
	return c.global.LookupRule(from)
}

// parseDirectives is called in each directory visited by gazelle.  The relative
// directory name is given by 'rel' and the list of directives in the BUILD file
// are specified by 'directives'.
func (c *scalaConfig) parseDirectives(directives []rule.Directive) (err error) {
	for _, d := range directives {
		// log.Printf("parsing directive rel=%q, key=%q, value=%q", rel, d.Key, d.Value)
		switch d.Key {
		case ruleDirective:
			err = c.parseRuleDirective(d)
			if err != nil {
				return fmt.Errorf("parse %v: %w", d, err)
			}
		case overrideDirective:
			c.parseOverrideDirective(d)
		case indirectDependencyDirective:
			c.parseIndirectDependencyDirective(d)
		case implicitImportDirective:
			c.parseImplicitImportDirective(d)
		case scalaExplainDependencies:
			c.parseScalaExplainDependencies(d)
		case mapKindImportNameDirective:
			c.parseMapKindImportNameDirective(d)
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
		return fmt.Errorf("invalid scala_rule directive %+v: %w", d, err)
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
	// o.dep = o.dep.Abs("", c.rel) // TODO(pcj): this is really needed?
	c.overrides = append(c.overrides, &o)
}

func (c *scalaConfig) parseIndirectDependencyDirective(d rule.Directive) {
	parts := strings.Fields(d.Value)
	if len(parts) < 3 {
		log.Printf("invalid gazelle:indirect-dependency directive: expected 3+ parts, got %d (%v)", len(parts), parts)
		return
	}
	c.indirects = append(c.indirects, &indirectDependencySpec{
		lang: parts[0],
		imp:  parts[1],
		deps: parts[2:],
	})
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

func (c *scalaConfig) parseScalaExplainDependencies(d rule.Directive) {
	parts := strings.Fields(d.Value)
	if len(parts) != 1 {
		log.Printf("invalid gazelle:scala_explain_dependencies directive: expected 1+ parts, got %d (%v)", len(parts), parts)
		return
	}
	c.explainDependencies = parts[0] == "true"
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

func (c *scalaConfig) GetIndirectDependencies(lang, imp string) (deps []string) {
	dbg := false
	if dbg {
		log.Println("checking indirect deps", imp, len(c.indirects))
	}
	for _, d := range c.indirects {
		if d.lang != lang {
			continue
		}
		if d.imp != imp {
			continue
		}
		deps = append(deps, d.deps...)
	}
	if dbg {
		log.Println("indirect:", imp, deps)
	}
	return
}

func (c *scalaConfig) GetImplicitImports(lang, imp string) (deps []string) {
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

func (c *scalaConfig) Overrides() []*overrideSpec {
	return c.overrides
}

func (c *scalaConfig) Indirects() []*indirectDependencySpec {
	return c.indirects
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

type indirectDependencySpec struct {
	// lang is the language to which this indirect applies.  Always 'scala' for now.
	lang string
	// imp is the "source" dependency (e.g. LazyLogging)
	imp string
	// dep is the "destination" dependencies (e.g. org.slf4j.Logger)
	deps []string
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
