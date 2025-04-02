package scalarule

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/rs/zerolog"

	"github.com/stackb/scala-gazelle/pkg/collections"
)

// Config carries metadata about a rule and its dependencies.
type Config struct {
	// Rel is the relative directory of this rule config
	Rel string
	// Config is the parent gazelle Config
	Config *config.Config
	// Deps is a mapping from label to +/- intent.
	Deps map[string]bool
	// Attr is a mapping from string to intent map.
	Attrs map[string]map[string]bool
	// Options is a generic key -> value string mapping.  Various rule
	// implementations are free to document/interpret options in an
	// implementation-dependenct manner.
	Options map[string]bool
	// Enabled is a flag that marks language generation as enabled or not
	Enabled bool
	// Implementation is the registry identifier for the provider
	Implementation string
	// Provider is the actual implementation
	Provider Provider
	// Name is the name of the Rule config
	Name string
	// Logger is a logger instance that can be used for debugging.
	Logger zerolog.Logger
}

// NewConfig returns a pointer to a new Config config with the
// 'Enabled' bit set to true.
func NewConfig(logger zerolog.Logger, config *config.Config, name string) *Config {
	return &Config{
		Logger:  logger,
		Config:  config,
		Name:    name,
		Enabled: true,
		Attrs:   make(map[string]map[string]bool),
		Deps:    make(map[string]bool),
		Options: make(map[string]bool),
	}
}

// GetDeps returns the sorted list of dependencies
func (c *Config) GetDeps() []string {
	deps := make([]string, 0)
	for dep, want := range c.Deps {
		if !want {
			continue
		}
		deps = append(deps, dep)
	}
	sort.Strings(deps)
	return deps
}

// GetOptions returns the rule options.
func (c *Config) GetOptions() []string {
	opts := make([]string, 0)
	for opt, want := range c.Options {
		if !want {
			continue
		}
		opts = append(opts, opt)
	}
	sort.Strings(opts)
	return opts
}

// GetAttr returns the positive-intent attr values under the given key.
func (c *Config) GetAttr(name string) []string {
	vals := make([]string, 0)
	for val, want := range c.Attrs[name] {
		if !want {
			continue
		}
		vals = append(vals, val)
	}
	sort.Strings(vals)
	return vals
}

// Clone copies this config to a new one
func (c *Config) Clone() *Config {
	clone := NewConfig(c.Logger, c.Config, c.Name)
	clone.Enabled = c.Enabled
	clone.Implementation = c.Implementation

	for name, vals := range c.Attrs {
		clone.Attrs[name] = make(map[string]bool)
		for k, v := range vals {
			clone.Attrs[name][k] = v
		}
	}
	for k, v := range c.Deps {
		clone.Deps[k] = v
	}
	for k, v := range c.Options {
		clone.Options[k] = v
	}
	return clone
}

// ParseDirective parses the directive string or returns error.
func (c *Config) ParseDirective(d, param, value string) error {
	intent := collections.ParseIntent(param)
	switch intent.Value {
	case "dep", "deps":
		if intent.Want {
			c.Deps[value] = true
		} else {
			delete(c.Deps, value)
		}
	case "option":
		if intent.Want {
			c.Options[value] = true
		} else {
			delete(c.Options, value)
		}
	case "attr":
		kv := strings.Fields(value)
		if len(kv) == 0 {
			return fmt.Errorf("malformed attr (missing attr name and value) %q: expected form is 'gazelle:proto_rule {RULE_NAME} attr {ATTR_NAME} [+/-]{VALUE}'", value)
		}
		key := collections.ParseIntent(kv[0])

		if len(kv) == 1 {
			if intent.Want {
				return fmt.Errorf("malformed attr %q (missing named attr value): expected form is 'gazelle:proto_rule {RULE_NAME} attr {ATTR_NAME} [+/-]{VALUE}'", value)
			} else {
				delete(c.Attrs, key.Value)
				return nil
			}
		}

		val := strings.Join(kv[1:], " ")

		if intent.Want {
			values, ok := c.Attrs[key.Value]
			if !ok {
				values = make(map[string]bool)
				c.Attrs[key.Value] = values
			}
			values[val] = key.Want
		} else {
			delete(c.Attrs, key.Value)
		}
	case "implementation":
		c.Implementation = value
	case "enabled":
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("enabled %s: %w", value, err)
		}
		c.Enabled = enabled
	default:
		return fmt.Errorf("unknown parameter %q", intent.Value)
	}

	return nil
}
